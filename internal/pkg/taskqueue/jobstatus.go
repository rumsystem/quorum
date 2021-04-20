package taskqueue

import "strconv"

const _jobStatus_name = "pendinginprogresscompletefailed"

var _jobStatus_index = [...]uint8{0, 7, 17, 25, 31}

func (i jobStatus) String() string {
	if i >= jobStatus(len(_jobStatus_index)-1) {
		return "jobStatus(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _jobStatus_name[_jobStatus_index[i]:_jobStatus_index[i+1]]
}
