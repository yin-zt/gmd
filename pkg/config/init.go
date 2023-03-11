package config

type Config struct {
	DefaultModule string
	DefaultAction string
	ScriptPath    string
	Salt          string
	Args          []string
	EtcdConf      *EtcdConf
	ShellStr      string
	Commands      chan interface{}
	Indexs        []int64
	_Args         string
	_ArgsSep      string
	UUID          string
	IP            string
}

type EtcdConf struct {
	User     string   `json:"user"`
	Password string   `json:"password"`
	Server   []string `json:"server"`
	Prefix   string   `json:"prefix"`
}

const (
	CONST_VERSION = "1.0-20230102"
	LogConfigStr  = `
<seelog type="asynctimer" asyncinterval="1000" minlevel="trace" maxlevel="error">  
	<outputs formatid="common">  
		<buffered formatid="common" size="1048576" flushperiod="1000">  
			<rollingfile type="size" filename="/var/log/gmd.log" maxsize="104857600" maxrolls="10"/>  
		</buffered>
	</outputs>  	  
	 <formats>
		 <format id="common" format="%Date %Time [%LEV] [%File:%Line] [%Func] %Msg%n" />  
	 </formats>  
</seelog>
`
)

var (
	DEBUG = false
)
