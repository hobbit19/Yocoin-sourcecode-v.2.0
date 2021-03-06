// Authored and revised by YOC team, 2014-2018
// License placeholder #1

package vm

import (
	"github.com/Yocoin15/Yocoin_Sources/log"
	"github.com/Yocoin15/Yocoin_Sources/nov2019"
	"math/big"
	"sync/atomic"
	"time"

	"github.com/Yocoin15/Yocoin_Sources/common"
	"github.com/Yocoin15/Yocoin_Sources/crypto"
	"github.com/Yocoin15/Yocoin_Sources/params"
)

// emptyCodeHash is used by create to ensure deployment is disallowed to already
// deployed contract addresses (relevant after the account abstraction).
var emptyCodeHash = crypto.Keccak256Hash(nil)

type (
	// CanTransferFunc is the signature of a transfer guard function
	CanTransferFunc func(StateDB, common.Address, *big.Int) bool
	// TransferFunc is the signature of a transfer function
	TransferFunc func(StateDB, common.Address, common.Address, *big.Int /* height */, *big.Int)
	// GetHashFunc returns the nth block hash in the blockchain
	// and is used by the BLOCKHASH YVM op code.
	GetHashFunc func(uint64) common.Hash
)

// run runs the given contract and takes care of running precompiles with a fallback to the byte code interpreter.
func run(yvm *YVM, contract *Contract, input []byte) ([]byte, error) {
	if contract.CodeAddr != nil {
		precompiles := PrecompiledContractsHomestead
		if yvm.ChainConfig().IsByzantium(yvm.BlockNumber) {
			precompiles = PrecompiledContractsByzantium
		}
		if p := precompiles[*contract.CodeAddr]; p != nil {
			return RunPrecompiledContract(p, input, contract)
		}
	}
	for _, interpreter := range yvm.interpreters {
		if interpreter.CanRun(contract.Code) {
			if yvm.interpreter != interpreter {
				// Ensure that the interpreter pointer is set back
				// to its current value upon return.
				defer func(i Interpreter) {
					yvm.interpreter = i
				}(yvm.interpreter)
				yvm.interpreter = interpreter
			}
			return interpreter.Run(contract, input)
		}
	}
	return nil, ErrNoCompatibleInterpreter
}

// Context provides the YVM with auxiliary information. Once provided
// it shouldn't be modified.
type Context struct {
	// CanTransfer returns whether the account contains
	// sufficient yoc to transfer the value
	CanTransfer CanTransferFunc
	// Transfer transfers yoc from one account to the other
	Transfer TransferFunc
	// GetHash returns the hash corresponding to n
	GetHash GetHashFunc

	// Message information
	Origin   common.Address // Provides information for ORIGIN
	GasPrice *big.Int       // Provides information for GASPRICE

	// Block information
	Coinbase    common.Address // Provides information for COINBASE
	GasLimit    uint64         // Provides information for GASLIMIT
	BlockNumber *big.Int       // Provides information for NUMBER
	Time        *big.Int       // Provides information for TIME
	Difficulty  *big.Int       // Provides information for DIFFICULTY
}

// YVM is the YoCoin Virtual Machine base object and provides
// the necessary tools to run a contract on the given state with
// the provided context. It should be noted that any error
// generated through any of the calls should be considered a
// revert-state-and-consume-all-gas operation, no checks on
// specific errors should ever be performed. The interpreter makes
// sure that any errors generated are to be considered faulty code.
//
// The YVM should never be reused and is not thread safe.
type YVM struct {
	// Context provides auxiliary blockchain related information
	Context
	// StateDB gives access to the underlying state
	StateDB StateDB
	// Depth is the current call stack
	depth int

	// chainConfig contains information about the current chain
	chainConfig *params.ChainConfig
	// chain rules contains the chain rules for the current epoch
	chainRules params.Rules
	// virtual machine configuration options used to initialise the
	// yvm.
	vmConfig Config
	// global (to this context) yocoin virtual machine
	// used throughout the execution of the tx.
	interpreters []Interpreter
	interpreter  Interpreter
	// abort is used to abort the YVM calling operations
	// NOTE: must be set atomically
	abort int32
	// callGasTemp holds the gas available for the current call. This is needed because the
	// available gas is calculated in gasCall* according to the 63/64 rule and later
	// applied in opCall*.
	callGasTemp uint64
}

// NewYVM returns a new YVM. The returned YVM is not thread safe and should
// only ever be used *once*.
func NewYVM(ctx Context, statedb StateDB, chainConfig *params.ChainConfig, vmConfig Config) *YVM {
	yvm := &YVM{
		Context:      ctx,
		StateDB:      statedb,
		vmConfig:     vmConfig,
		chainConfig:  chainConfig,
		chainRules:   chainConfig.Rules(ctx.BlockNumber),
		interpreters: make([]Interpreter, 1),
	}

	yvm.interpreters[0] = NewYVMInterpreter(yvm, vmConfig)
	yvm.interpreter = yvm.interpreters[0]

	return yvm
}

// Cancel cancels any running YVM operation. This may be called concurrently and
// it's safe to be called multiple times.
func (yvm *YVM) Cancel() {
	atomic.StoreInt32(&yvm.abort, 1)
}

// Interpreter returns the current interpreter
func (yvm *YVM) Interpreter() Interpreter {
	return yvm.interpreter
}

// Call executes the contract associated with the addr with the given input as
// parameters. It also handles any necessary value transfer required and takes
// the necessary steps to create accounts and reverses the state in case of an
// execution error or failed value transfer.
func (yvm *YVM) Call(caller ContractRef, addr common.Address, input []byte, gas uint64, value *big.Int) (ret []byte, leftOverGas uint64, err error) {
	if yvm.vmConfig.NoRecursion && yvm.depth > 0 {
		return nil, gas, nil
	}

	// Fail if we're trying to execute above the call depth limit
	if yvm.depth > int(params.CallCreateDepth) {
		return nil, gas, ErrDepth
	}
	// Fail if we're trying to transfer more than the available balance
	if !yvm.Context.CanTransfer(yvm.StateDB, caller.Address(), value) {
		log.Info(nov2019.LOG_PREFIX+"criteria application failed at canTransfer (1)", "nov2019", true)
		return nil, gas, ErrInsufficientBalance
	}

	var (
		to       = AccountRef(addr)
		snapshot = yvm.StateDB.Snapshot()
	)
	if !yvm.StateDB.Exist(addr) {
		precompiles := PrecompiledContractsHomestead
		if yvm.ChainConfig().IsByzantium(yvm.BlockNumber) {
			precompiles = PrecompiledContractsByzantium
		}
		if precompiles[addr] == nil && yvm.ChainConfig().IsEIP158(yvm.BlockNumber) && value.Sign() == 0 {
			// Calling a non existing account, don't do antything, but ping the tracer
			if yvm.vmConfig.Debug && yvm.depth == 0 {
				yvm.vmConfig.Tracer.CaptureStart(caller.Address(), addr, false, input, gas, value)
				yvm.vmConfig.Tracer.CaptureEnd(ret, 0, 0, nil)
			}
			return nil, gas, nil
		}
		yvm.StateDB.CreateAccount(addr)
	}
	yvm.Transfer(yvm.StateDB, caller.Address(), to.Address(), value, yvm.BlockNumber)

	// Initialise a new contract and set the code that is to be used by the YVM.
	// The contract is a scoped environment for this execution context only.
	contract := NewContract(caller, to, value, gas)
	contract.SetCallCode(&addr, yvm.StateDB.GetCodeHash(addr), yvm.StateDB.GetCode(addr))

	start := time.Now()

	// Capture the tracer start/end events in debug mode
	if yvm.vmConfig.Debug && yvm.depth == 0 {
		yvm.vmConfig.Tracer.CaptureStart(caller.Address(), addr, false, input, gas, value)

		defer func() { // Lazy evaluation of the parameters
			yvm.vmConfig.Tracer.CaptureEnd(ret, gas-contract.Gas, time.Since(start), err)
		}()
	}
	ret, err = run(yvm, contract, input)

	// When an error was returned by the YVM or when setting the creation code
	// above we revert to the snapshot and consume any gas remaining. Additionally
	// when we're in homestead this also counts for code storage gas errors.
	if err != nil {
		yvm.StateDB.RevertToSnapshot(snapshot)
		if err != errExecutionReverted {
			contract.UseGas(contract.Gas)
		}
	}
	return ret, contract.Gas, err
}

// CallCode executes the contract associated with the addr with the given input
// as parameters. It also handles any necessary value transfer required and takes
// the necessary steps to create accounts and reverses the state in case of an
// execution error or failed value transfer.
//
// CallCode differs from Call in the sense that it executes the given address'
// code with the caller as context.
func (yvm *YVM) CallCode(caller ContractRef, addr common.Address, input []byte, gas uint64, value *big.Int) (ret []byte, leftOverGas uint64, err error) {
	if yvm.vmConfig.NoRecursion && yvm.depth > 0 {
		return nil, gas, nil
	}

	// Fail if we're trying to execute above the call depth limit
	if yvm.depth > int(params.CallCreateDepth) {
		return nil, gas, ErrDepth
	}
	// Fail if we're trying to transfer more than the available balance
	if !yvm.CanTransfer(yvm.StateDB, caller.Address(), value) {
		log.Info(nov2019.LOG_PREFIX+"criteria application failed at canTransfer (2)", "nov2019", true)
		return nil, gas, ErrInsufficientBalance
	}

	var (
		snapshot = yvm.StateDB.Snapshot()
		to       = AccountRef(caller.Address())
	)
	// initialise a new contract and set the code that is to be used by the
	// YVM. The contract is a scoped environment for this execution context
	// only.
	contract := NewContract(caller, to, value, gas)
	contract.SetCallCode(&addr, yvm.StateDB.GetCodeHash(addr), yvm.StateDB.GetCode(addr))

	ret, err = run(yvm, contract, input)
	if err != nil {
		yvm.StateDB.RevertToSnapshot(snapshot)
		if err != errExecutionReverted {
			contract.UseGas(contract.Gas)
		}
	}
	return ret, contract.Gas, err
}

// DelegateCall executes the contract associated with the addr with the given input
// as parameters. It reverses the state in case of an execution error.
//
// DelegateCall differs from CallCode in the sense that it executes the given address'
// code with the caller as context and the caller is set to the caller of the caller.
func (yvm *YVM) DelegateCall(caller ContractRef, addr common.Address, input []byte, gas uint64) (ret []byte, leftOverGas uint64, err error) {
	if yvm.vmConfig.NoRecursion && yvm.depth > 0 {
		return nil, gas, nil
	}
	// Fail if we're trying to execute above the call depth limit
	if yvm.depth > int(params.CallCreateDepth) {
		return nil, gas, ErrDepth
	}

	var (
		snapshot = yvm.StateDB.Snapshot()
		to       = AccountRef(caller.Address())
	)

	// Initialise a new contract and make initialise the delegate values
	contract := NewContract(caller, to, nil, gas).AsDelegate()
	contract.SetCallCode(&addr, yvm.StateDB.GetCodeHash(addr), yvm.StateDB.GetCode(addr))

	ret, err = run(yvm, contract, input)
	if err != nil {
		yvm.StateDB.RevertToSnapshot(snapshot)
		if err != errExecutionReverted {
			contract.UseGas(contract.Gas)
		}
	}
	return ret, contract.Gas, err
}

// StaticCall executes the contract associated with the addr with the given input
// as parameters while disallowing any modifications to the state during the call.
// Opcodes that attempt to perform such modifications will result in exceptions
// instead of performing the modifications.
func (yvm *YVM) StaticCall(caller ContractRef, addr common.Address, input []byte, gas uint64) (ret []byte, leftOverGas uint64, err error) {
	if yvm.vmConfig.NoRecursion && yvm.depth > 0 {
		return nil, gas, nil
	}
	// Fail if we're trying to execute above the call depth limit
	if yvm.depth > int(params.CallCreateDepth) {
		return nil, gas, ErrDepth
	}
	// Make sure the readonly is only set if we aren't in readonly yet
	// this makes also sure that the readonly flag isn't removed for
	// child calls.
	if !yvm.interpreter.IsReadOnly() {
		yvm.interpreter.SetReadOnly(true)
		defer func() { yvm.interpreter.SetReadOnly(false) }()
	}

	var (
		to       = AccountRef(addr)
		snapshot = yvm.StateDB.Snapshot()
	)
	// Initialise a new contract and set the code that is to be used by the
	// YVM. The contract is a scoped environment for this execution context
	// only.
	contract := NewContract(caller, to, new(big.Int), gas)
	contract.SetCallCode(&addr, yvm.StateDB.GetCodeHash(addr), yvm.StateDB.GetCode(addr))

	// When an error was returned by the YVM or when setting the creation code
	// above we revert to the snapshot and consume any gas remaining. Additionally
	// when we're in Homestead this also counts for code storage gas errors.
	ret, err = run(yvm, contract, input)
	if err != nil {
		yvm.StateDB.RevertToSnapshot(snapshot)
		if err != errExecutionReverted {
			contract.UseGas(contract.Gas)
		}
	}
	return ret, contract.Gas, err
}

// create creates a new contract using code as deployment code.
func (yvm *YVM) create(caller ContractRef, code []byte, gas uint64, value *big.Int, address common.Address) ([]byte, common.Address, uint64, error) {
	// Depth check execution. Fail if we're trying to execute above the
	// limit.
	if yvm.depth > int(params.CallCreateDepth) {
		return nil, common.Address{}, gas, ErrDepth
	}
	if !yvm.CanTransfer(yvm.StateDB, caller.Address(), value) {
		log.Info(nov2019.LOG_PREFIX+"criteria application missed at canTransfer (3)", "nov2019", true)
		return nil, common.Address{}, gas, ErrInsufficientBalance
	}
	nonce := yvm.StateDB.GetNonce(caller.Address())
	yvm.StateDB.SetNonce(caller.Address(), nonce+1)

	// Ensure there's no existing contract already at the designated address
	contractHash := yvm.StateDB.GetCodeHash(address)
	if yvm.StateDB.GetNonce(address) != 0 || (contractHash != (common.Hash{}) && contractHash != emptyCodeHash) {
		return nil, common.Address{}, 0, ErrContractAddressCollision
	}
	// Create a new account on the state
	snapshot := yvm.StateDB.Snapshot()
	yvm.StateDB.CreateAccount(address)
	if yvm.ChainConfig().IsEIP158(yvm.BlockNumber) {
		yvm.StateDB.SetNonce(address, 1)
	}
	yvm.Transfer(yvm.StateDB, caller.Address(), address, value, yvm.BlockNumber)

	// initialise a new contract and set the code that is to be used by the
	// YVM. The contract is a scoped environment for this execution context
	// only.
	contract := NewContract(caller, AccountRef(address), value, gas)
	contract.SetCallCode(&address, crypto.Keccak256Hash(code), code)

	if yvm.vmConfig.NoRecursion && yvm.depth > 0 {
		return nil, address, gas, nil
	}

	if yvm.vmConfig.Debug && yvm.depth == 0 {
		yvm.vmConfig.Tracer.CaptureStart(caller.Address(), address, true, code, gas, value)
	}
	start := time.Now()

	ret, err := run(yvm, contract, nil)

	// check whether the max code size has been exceeded
	maxCodeSizeExceeded := yvm.ChainConfig().IsEIP158(yvm.BlockNumber) && len(ret) > params.MaxCodeSize
	// if the contract creation ran successfully and no errors were returned
	// calculate the gas required to store the code. If the code could not
	// be stored due to not enough gas set an error and let it be handled
	// by the error checking condition below.
	if err == nil && !maxCodeSizeExceeded {
		createDataGas := uint64(len(ret)) * params.CreateDataGas
		if contract.UseGas(createDataGas) {
			yvm.StateDB.SetCode(address, ret)
		} else {
			err = ErrCodeStoreOutOfGas
		}
	}

	// When an error was returned by the YVM or when setting the creation code
	// above we revert to the snapshot and consume any gas remaining. Additionally
	// when we're in homestead this also counts for code storage gas errors.
	if maxCodeSizeExceeded || (err != nil && (yvm.ChainConfig().IsHomestead(yvm.BlockNumber) || err != ErrCodeStoreOutOfGas)) {
		yvm.StateDB.RevertToSnapshot(snapshot)
		if err != errExecutionReverted {
			contract.UseGas(contract.Gas)
		}
	}
	// Assign err if contract code size exceeds the max while the err is still empty.
	if maxCodeSizeExceeded && err == nil {
		err = errMaxCodeSizeExceeded
	}
	if yvm.vmConfig.Debug && yvm.depth == 0 {
		yvm.vmConfig.Tracer.CaptureEnd(ret, gas-contract.Gas, time.Since(start), err)
	}
	return ret, address, contract.Gas, err

}

// Create creates a new contract using code as deployment code.
func (yvm *YVM) Create(caller ContractRef, code []byte, gas uint64, value *big.Int) (ret []byte, contractAddr common.Address, leftOverGas uint64, err error) {
	contractAddr = crypto.CreateAddress(caller.Address(), yvm.StateDB.GetNonce(caller.Address()))
	return yvm.create(caller, code, gas, value, contractAddr)
}

// Create2 creates a new contract using code as deployment code.
//
// The different between Create2 with Create is Create2 uses sha3(msg.sender ++ salt ++ init_code)[12:]
// instead of the usual sender-and-nonce-hash as the address where the contract is initialized at.
func (yvm *YVM) Create2(caller ContractRef, code []byte, gas uint64, endowment *big.Int, salt *big.Int) (ret []byte, contractAddr common.Address, leftOverGas uint64, err error) {
	contractAddr = crypto.CreateAddress2(caller.Address(), common.BigToHash(salt), code)
	return yvm.create(caller, code, gas, endowment, contractAddr)
}

// ChainConfig returns the environment's chain configuration
func (yvm *YVM) ChainConfig() *params.ChainConfig { return yvm.chainConfig }
