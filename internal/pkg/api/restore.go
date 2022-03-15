package api

type BackupResult struct {
	// encrypt json.Marshal([]GroupSeed)
	Seeds    string `json:"seeds"`
	Keystore string `json:"keystore"`
	Config   string `json:"config" validate:"required"`
}
