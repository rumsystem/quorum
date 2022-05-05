package def

type GroupMgrIface interface {
	GetGroup(groupId string) (GroupIface, error)
}
