package kubemgr

type KubeMgr struct {
	filePath string
}

func NewKubeMgr(filePath string) *KubeMgr {
	k := KubeMgr{}
	k.filePath = filePath
	return &k
}

func (k *KubeMgr) Do(action string, target string) {
	config := k.getSaneConfig()

	switch action {
	case ActionApply:
		config.Apply(target)
	case ActionCheck:
		config.Check(target)
	case ActionDelete:
		config.Delete(target)
	}
}

func (k *KubeMgr) Apply(target string) {
	k.getSaneConfig().Apply(target)
}

func (k *KubeMgr) Check(target string) {
	k.getSaneConfig().Check(target)

}

func (k *KubeMgr) Delete(target string) {
	k.getSaneConfig().Delete(target)
}

func (k *KubeMgr) getSaneConfig() *SaneConfig {
	return NewRawConfig().FromFilePath(k.filePath).Sanitize()
}
