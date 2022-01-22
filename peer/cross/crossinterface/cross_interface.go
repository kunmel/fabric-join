package crossinterface

type CrossInterface interface {
	CrossConfirmSucc(txid string) error
	CrossConfirmFail(channelid string, txid string) error
}

type CrossHandler struct{
	Itfc	CrossInterface
}






