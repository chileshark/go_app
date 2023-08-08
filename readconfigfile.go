package main

import (
	"flag"
	"fmt"
	probing "github.com/prometheus-community/pro-bing"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
	"gopkg.in/ini.v1"
	"log"
	"time"
)

type PushGatewayINIConfig struct {
	PushGateWayAddress string `ini:"PushGateWayAddress"`
	MetricKey          string `ini:"MetricKey"`
	JobKey             string `ini:"JobKey"`
}

type DestinationINIConfig struct {
	DestinationRegion string `ini:"DestinationRegion"`
	DestinationIp     string `ini:"DestinationIp"`
	DestinationAZ     string `ini:"DestinationAZ"`
}

type SourceINIConfig struct {
	SourceRegion string `ini:"SourceRegion"`
	SourceAZ     string `ini:"SourceAZ"`
	PingSpeed    int    `ini:"PingSpeed"`
	PingCount    int    `ini:"PingCount"`
}
type INIconfig struct {
	PushGateway    PushGatewayINIConfig `ini:"PushGateWays"`
	DestinationMsg DestinationINIConfig `ini:"DestinationMsg"`
	SourceMsg      SourceINIConfig      `ini:"SourceMsg"`
}

// Read ConfigFile
func LoadINIConfig(path string) *INIconfig {
	config := &INIconfig{}
	cfg, err := ini.Load(path)
	if err != nil {
		fmt.Println("配置文件不存在，请检查路径是否正确:", err)
		panic(err)
	}
	config.PushGateway.PushGateWayAddress = cfg.Section("PushGateWays").Key("PushGateWayAddress").String()
	config.PushGateway.MetricKey = cfg.Section("PushGateWays").Key("MetricKey").String()
	config.PushGateway.JobKey = cfg.Section("PushGateWays").Key("JobKey").String()

	config.DestinationMsg.DestinationRegion = cfg.Section("DestinationMsg").Key("DestinationRegion").String()
	config.DestinationMsg.DestinationIp = cfg.Section("DestinationMsg").Key("DestinationIp").String()
	config.DestinationMsg.DestinationAZ = cfg.Section("DestinationMsg").Key("DestinationAZ").String()

	config.SourceMsg.SourceAZ = cfg.Section("SourceMsg").Key("SourceAZ").String()
	config.SourceMsg.SourceRegion = cfg.Section("SourceMsg").Key("SourceRegion").String()
	config.SourceMsg.PingSpeed, _ = cfg.Section("SourceMsg").Key("PingSpeed").Int()
	config.SourceMsg.PingCount, _ = cfg.Section("SourceMsg").Key("PingCount").Int()
	return config
}

// Send ICMP Package
func Pingaddress(addr string) (float64, float64, float64, float64) {
	pinger, err := probing.NewPinger(addr)
	if err != nil {
		panic(err)
	}
	pinger.Count = 5
	pinger.Timeout = time.Second * 10
	pinger.SetPrivileged(true)
	err = pinger.Run()
	if err != nil {
		panic(err)
	}
	stats := pinger.Statistics()
	if stats.PacketsRecv <= 0 {
		panic(stats)
	}
	// converts milliseconds to seconds
	MinRTT := stats.MinRtt.Seconds() * 1000
	AvgRtt := stats.AvgRtt.Seconds() * 1000
	MaxRtt := stats.MaxRtt.Seconds() * 1000

	LostPackagePercent := stats.PacketLoss
	//fmt.Printf("\nPing %v\nMinRTT is %v\nAvgRTT is %v\nMaxRTT is %v\nLostPackagePercent is %v\n", pinger.Addr(), MinRTT, AvgRtt, MaxRtt, LostPackagePercent)
	return MinRTT, AvgRtt, MaxRtt, LostPackagePercent
}

var (
	PingLatencyInfoBetweenRegionsLatency = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "PingLatencyInfoBetweenRegionsLatency",
		Help: "PingLatencyInfoBetweenRegionsLatency",
	}, []string{"Latency"})
)

func Pushgateways(Targetaddress string, PushGatewayAddr string, Keys string, SourceRegion string, DestinationRegion string, SourceAZ string, DestinationAZ string) {
	MinRtt, AvgRtt, MaxRtt, LostPackagePercent := Pingaddress(Targetaddress)
	PingLatencyInfoBetweenRegionsLatency.WithLabelValues("Min").Set(MinRtt)
	PingLatencyInfoBetweenRegionsLatency.WithLabelValues("Avg").Set(AvgRtt)
	PingLatencyInfoBetweenRegionsLatency.WithLabelValues("Max").Set(MaxRtt)
	PingLatencyInfoBetweenRegionsLatency.WithLabelValues("LostPackagePercent").Set(LostPackagePercent)
	pusher := push.New(PushGatewayAddr, Keys).Collector(PingLatencyInfoBetweenRegionsLatency).Grouping("SourceRegion", SourceRegion).Grouping("DestinationRegion", DestinationRegion).Grouping("SourceAZ", SourceAZ).
		Grouping("DestinationAZ", DestinationAZ).Grouping("instance", "1.1.1.1").Push()
	if pusher != nil {
		panic(pusher)
	}
}

func main() {
	PingCount := 0
	var configPath string
	flag.StringVar(&configPath, "a", "", "ConfigFilePath")
	flag.Parse()
	confINI := LoadINIConfig(configPath)

	DestinationIP := confINI.DestinationMsg.DestinationIp
	DestinationRegion := confINI.DestinationMsg.DestinationRegion
	DestinationAZ := confINI.DestinationMsg.DestinationAZ

	Pushgateway := confINI.PushGateway.PushGateWayAddress
	JobKey := confINI.PushGateway.MetricKey

	SourceRegion := confINI.SourceMsg.SourceRegion
	SourceAZ := confINI.SourceMsg.SourceAZ
	SourcePingCount := confINI.SourceMsg.PingCount
	SourcePingSpeed := confINI.SourceMsg.PingSpeed
	for {
		Pushgateways(DestinationIP, Pushgateway, JobKey, SourceRegion, DestinationRegion, SourceAZ, DestinationAZ)
		PingCount += 1
		log.Printf("第%v次ping地址%s,速率为%v秒一次,一次Ping%v个包", PingCount, DestinationIP, SourcePingSpeed, SourcePingCount)
		time.Sleep(30 * time.Second)
	}
}
