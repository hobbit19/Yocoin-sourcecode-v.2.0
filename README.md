# YoCoin


## Building

Golang 1.9+ required.
On Ubuntu 16.04+:
    
    sudo add-apt-repository ppa:gophers/archive
    sudo apt update
    sudo apt install golang-1.9-go

Verify installation:

    go version

Then:

    mkdir -p ~/go/src/github.com/Yocoin15/
    rm -rf ~/go/src/github.com/Yocoin15/Yocoin_Sources
    git clone git@github.com:Yocoin15/Yocoin-sourcecode-v.2.0.git 
    export GOPATH=~/go
    cd ~/go/src/github.com/Yocoin15/Yocoin_Sources
    go build -o ./yocoin cmd/yocoin/*.go

### Building bundled JS

    cd internal/jsre/deps/
    go get -u github.com/jteeuwen/go-bindata/...
    go generate
    cd ../../..
    go build -o ./yocoin cmd/yocoin/*.go

($PATH must include ~/go/bin for go-bindata tool to be available)

*Hint: preprocess .js first with https://closure-compiler.appspot.com/ ("whitespace only" _ "pretty print")*

### Building debian package

Tnis approach is using debian/ template - not build/ci.go script (which was used to generate the template)

    sudo apt install devscripts debhelper dh-systemd
    dpkg-buildpackage

Your production system should have GPG key for `"YoCoin Builds <yocoin@yocoin.org>"`

## Starting node and testnet

Create new account(-s)

     ./yocoin account new

Create genesis.json file from genesis.example.json and add new account to "alloc" block. To initialize private network:

    ./yocoin init genesis.json

Run new node in "console" mode
 
    build/bin/yoc --maxpeers 0 --networkid 13 --port "30301" --rpc --rpccorsdomain "*" console

Check mining base address (console)

    > yoc.coinbase

Start mining to generate blocks (console)

    > miner.start()

## Working with node

First, run

    ./yocoin account new

To list accounts:

    ./yocoin account list

Console mode:

    ./yocoin console

Unlock your account (console):

    > personal.unlockAccount("account_address")

With default account (console):

    > personal.unlockAccount(yoc.accounts[0])

With password and unlock timeout (console):

    > personal.unlockAccount("account_address", "password", 300)

Sending (console, value in 1/1000000000 YOCs):

     > yoc.sendTransaction({from:yoc.accounts[0], to:"0x5adc90e0637eb8bf0f5022611214ee500afae06d", value:"17590519640"});

Getting balance:

    > web3.fromWei(yoc.getBalance(yoc.coinbase), "gwei")

## Node RPC


to run rpc methods over local HTTP connection, use --rpc:

    ./yocoin --rpc

then do queries like in go-ethereum:

    curl -X POST -H "Content-Type: application/json" --data '{"jsonrpc":"2.0","method":"yoc_accounts","params":[],"id":1} http://127.0.0.1:8545'

Starting testnet node accessible from outside and able to communicate with more nodes than on fixed list (--discover):

    ./yocoin --networkid 7357 --rpccorsdomain "*" --rpc --rpcport 8545 --rpcaddr "0.0.0.0" --discover console

Limiting exposed APIs:

    ./yocoin --rpcapi yoc,web3,net,personal --rpc --rpcport "8545" --rpcaddr "0.0.0.0" --rpccorsdomain "*"
 
Get balance:

    curl -s -H "Content-Type: application/json" -X POST --data '{"jsonrpc":"2.0", "method":"yoc_getBalance", "params":["0x5adc90e0637eb8bf0f5022611214ee500afae06d", "latest"], "id":1}' http://localhost:8545

Send transaction:

    curl -s -H "Content-Type: application/json" -X POST --data '{"jsonrpc":"2.0", "method":"yoc_sendTransaction", "params":[{"from": "0x5adc90e0637eb8bf0f5022611214ee500afae06d", "to": "0x5adc90e0637eb8bf0f5022611214ee500afae06d", "value": "0x0"}], "id":1}' http://localhost:8545

Value is in gwei (1/1000000000 YOC), and is HEX-encoded. (For reference, 0.1 YOC is 0x5F5E100) 


YoCoin node is set by default to use tcp:8545 for application/json HTTP POST requets and tcp:8546 port for websocket requests.  id" can stay equal to "1", but providing at least the always same id is mandatory.



## More cURL examples


### Create account, unlock it and send from it

Create new account (with password):

    curl -k -s -H "Content-Type: application/json" -X POST --data '{"jsonrpc":"2.0", "method":"personal_newAccount", "params":["password111lol"],"id":1}' http://localhost:8545

Expected output is account address:

    {"jsonrpc":"2.0","id":1,"result":"0xa042d63e84d2fa86fc4f8aa69832cf3eca3d94a0"}

Now unlock this account so it can be used:

    curl -k -s -H "Content-Type: application/json" -X POST --data '{"jsonrpc":"2.0", "method":"personal_unlockAccount", "params": ["0xa042d63e84d2fa86fc4f8aa69832cf3eca3d94a0", "password111lol", 3000], "id": 1}'

Expected output is confirmation:

    {"jsonrpc":"2.0","id":1,"result":true}
   

Sending coins (when you get some on the account):

    curl -s -H "Content-Type: application/json" -X POST --data '{"jsonrpc":"2.0", "method":"yoc_sendTransaction", "params":[{"from": "0xa042d63e84d2fa86fc4f8aa69832cf3eca3d94a0", "to": "0xa042d63e84d2fa86fc4f8aa69832cf3eca3d94a0", "value": "0x5F5E100"}], "id":4}' http://localhost:8545

Value is in gwei (1/1000000000 YOC), and is HEX-encoded. (For reference, 0.1 YOC is 0x5F5E100)

## RPC Methods

Some of the method namespaces are disabled by default; see examples of using --rpcapi above.

All methods are called either with array of arguments:

    curl -k -s -H "Content-Type: application/json" -X POST --data '{"jsonrpc":"2.0", "method":"NAME", "params":["ARRAY", "OF", "VARARGS"],"id":1}' http://localhost:8545

Or, with JSON object for argument:   

    curl -k -s -H "Content-Type: application/json" -X POST --data '{"jsonrpc":"2.0", "method":"NAME", "params":[{"JSON": "FIELD"],"id":1}' http://localhost:8545

"id" field is returned in response - useful when using same node for multiple requests which may be executed in variable order. Example response:

### personal_newAccount

* Requires 'personal' namespace to be enabled through --rpcapi option
* By default, creates an account with empty password
* Can takes a parameter: account password (plaintext)

### personal_unlockAccount
* Requires 'personal' namespace to be enabled through --rpcapi option
* "Unlocks" account so it can be used for sending funds
* Can take 1, 2 or 3 parameters: account address to unlock, unlock password and a timeout in seconds to keep account unlocked. Following variations can be used:
    * ["0xAddress"]
    * ["0xAddress", "password"]
    * ["0xAddress", "password", 10000]

### yoc_getBalance
Querying account balance.
* Requires 'yoc' namespace to be enabled through '--rpcapi' option
* Takes 1 or 2 parameters: account address to check balance, and balance type (use "latest")
    * ["0xAddress"] 
    * ["0xAddress", "latest"]
  * Returns a HEX value of account in 1/1000000000 YOC equivalent

Sample: ` curl -k -s -H "Content-Type: application/json" -X POST --data '{"jsonrpc":"2.0", "method":"yoc_getBalance", "params":["0xAddress"],"id":1}' http://localhost:8545`

### yoc_sendTransaction

* Required 'yoc' namespace to be enabled through '--rpcapi' option
* Sends YOC from one account to another
* Takes 1 JSON object parameter, containing other parameters
    * `[{"from": "0x5adc90e0637eb8bf0f5022611214ee500afae06d", "to": "0x5adc90e0637eb8bf0f5022611214ee500afae06d", "value": "0x5F5E100"}]`
        * "from": address , from which to withdraw
        * "to": receiving address (OPTIONAL if creating new contract)
        * "gas": (optional, default: 90000) Integer of the gas provided for the transaction execution. It will return unused gas.
        * "gasPrice": QUANTITY - (OPTIONAL, default: To-Be-Determined) Integer of the gasPrice used for each paid gas
        * "value": QUANTITY - (OPTIONAL) HEX-encoded integer of the value sent with this transaction. For YOC transaction, a value 
        * "data": DATA - The compiled code of a contract OR the hash of the invoked method signature and encoded parameters.
        * "nonce": QUANTITY - (OPTIONAL) Integer of a nonce. This allows to overwrite your own pending transactions that use the same nonce.

### yoc_getTransaction


* Returns info about transaction
* Takes one array parameter, transaction hash:
        * `["0xAddress"]`

## RPC minimal test

    ./yocoin --rpcapi yoc,web3,net,personal --rpc --rpcport "8546" --rpcaddr "0.0.0.0" --rpccorsdomain "*" account list
    ./yocoin --rpcapi yoc,web3,net,personal --rpc --rpcport "8546" --rpcaddr "0.0.0.0" --rpccorsdomain "*" --syncmode fast console
    
