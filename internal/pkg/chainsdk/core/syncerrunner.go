package chain

/*
func (sr *SyncerRunner) GetSyncCnsusTask(args ...interface{}) (*SyncTask, error) {
	syncerrunner_log.Debugf("<%s> GetConsensusSyncTask called", sr.groupId)
	taskmate := PSyncTask{SessionId: uuid.NewString()}
	return &SyncTask{TaskId: taskmate.SessionId, Type: PSync, RetryCount: 0, Meta: taskmate}, nil
}


func (sr *SyncerRunner) GetSyncLocalTask(args ...interface{}) (*SyncTask, error) {
	return nil, nil
}
*/

/*
else if task.Type == PSync {
		psynctask, ok := task.Meta.(PSyncTask)
		if !ok {
			gsyncer_log.Errorf("<%s> Unsupported task %s", sr.groupId, task.TaskId)
			return fmt.Errorf("<%s> Unsupported task %s", sr.groupId, task.TaskId)
		}

		syncerrunner_log.Debugf("<%s> TaskSender with PSync Task, SessionId <%s>", sr.groupId, psynctask.SessionId)

		//create psyncReqMsg
		psyncReqMsg := &quorumpb.PSyncReq{
			GroupId:      sr.groupId,
			SessionId:    psynctask.SessionId,
			SenderPubkey: sr.chainCtx.groupItem.UserSignPubkey,
			MyEpoch:      sr.chainCtx.GetCurrEpoch(),
		}

		//sign it
		bbytes, err := proto.Marshal(psyncReqMsg)
		if err != nil {
			return err
		}

		msgHash := localcrypto.Hash(bbytes)

		var signature []byte
		ks := localcrypto.GetKeystore()
		signature, err = ks.EthSignByKeyName(sr.groupId, msgHash, sr.nodename)

		if err != nil {
			return err
		}

		if len(signature) == 0 {
			return fmt.Errorf("create signature failed")
		}

		psyncReqMsg.SenderSign = signature

		payload, _ := proto.Marshal(psyncReqMsg)
		psyncMsg := &quorumpb.PSyncMsg{
			MsgType: quorumpb.PSyncMsgType_PSYNC_REQ,
			Payload: payload,
		}

		connMgr, err := conn.GetConn().GetConnMgr(sr.groupId)
		if err != nil {
			return err
		}

		err = connMgr.BroadcastPSyncMsg(psyncMsg)
		if err != nil {
			return err
		}

		return nil
	}

	return fmt.Errorf("<%s> Unsupported task type %s", sr.groupId, task.TaskId)
*/
