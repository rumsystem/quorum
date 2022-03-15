package storage

func CreateDb(path string) (*DbMgr, error) {
	var err error
	groupDb := QSBadger{}
	dataDb := QSBadger{}
	err = groupDb.Init(path + "_groups")
	if err != nil {
		return nil, err
	}

	err = dataDb.Init(path + "_db")
	if err != nil {
		return nil, err
	}

	manager := DbMgr{&groupDb, &dataDb, nil, path}
	return &manager, nil
}
