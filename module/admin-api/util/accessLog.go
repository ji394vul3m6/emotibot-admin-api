package util

type AccessLog struct {
	Path       string
	UserID     string
	UserIP     string
	AppID      string
	Time       float64
	Input      string
	Output     string
	StatusCode int
}

func InitAccessLog(channel chan AccessLog) {
	go func() {
		for {
			log := <-channel
			LogInfo.Printf("REQ: [%s][%d] [%.3fs][%s@%s]@[%s]",
				log.Path, log.StatusCode, log.Time, log.UserID, log.AppID, log.UserIP)
		}
	}()
}
