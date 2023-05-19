package handlers

type GetGroupCtnPrarms struct {
	GroupId         string   `param:"group_id" json:"group_id" url:"-" validate:"required,uuid4"`
	Num             int      `query:"num" json:"num" url:"num"`
	StartTrx        string   `query:"start_trx" json:"start_trx" url:"start_trx"`
	Reverse         bool     `query:"reverse" json:"reverse" url:"reverse,omitempty"`
	IncludeStartTrx bool     `query:"include_start_trx" json:"include_start_trx" url:"include_start_trx,omitempty"`
	Senders         []string `query:"senders" json:"senders" url:"senders"`
}
