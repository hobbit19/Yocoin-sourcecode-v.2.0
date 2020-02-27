package nov2019

// Балансы для тестнета с эспериментом по портированию клиента на новую сеть
const (
	nov2019RTBlock = "0x9d9b79d03d2584855d28573abd70c5f99fa1ae0c" // операции по этому балансу просто блокируются со входа (RPC, etc) в IfFilteringCriteria
	nov2019RTBlack = "0x72f3fe698694988061cd92008ff177d5c774011a" // этот баланс обнуляется в IfApplicableCriteria, ну и тоже блокируется в IfFilteringCriteria
	nov2019RTGood  = "0xfde53fa41cdfee341ff701a6402ca59d0c468f3d" // сюда восстанавливается баланс, сюда при включенной переменной идет майнинг
)
