// sysinfo_influxdb by Novaquark
//
// To the extent possible under law, the person who associated CC0 with
// sysinfo_influxdb has waived all copyright and related or neighboring rights
// to sysinfo_influxdb.
//
// You should have received a copy of the CC0 legalcode along with this
// work.  If not, see <http://creativecommons.org/publicdomain/zero/1.0/>.

package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"github.com/cloudfoundry/gosigar"
	influxClient "github.com/influxdb/influxdb-go"
)

const APP_VERSION = "0.1.0"


// Variables storing arguments flags
var verboseFlag bool
var versionFlag bool
var prefixFlag string

var hostFlag string
var usernameFlag string
var passwordFlag string
var databaseFlag string

func init() {
	flag.BoolVar(&versionFlag, "version", false, "Print the version number and exit.")
	flag.BoolVar(&versionFlag, "V", false, "Print the version number and exit (shorthand).")

	flag.BoolVar(&verboseFlag, "verbose", false, "Display debug information.")
	flag.BoolVar(&verboseFlag, "v", false, "Display debug information (shorthand).")

	hostname, _ := os.Hostname();
	flag.StringVar(&prefixFlag, "prefix", hostname, "Change series name prefix.")
	flag.StringVar(&prefixFlag, "P", hostname, "Change series name prefix (shorthand).")

	flag.StringVar(&hostFlag, "host", "localhost:8086", "Connect to host.")
	flag.StringVar(&hostFlag, "h", "localhost:8086", "Connect to host (shorthand).")
	flag.StringVar(&usernameFlag, "username", "root", "User for login.")
	flag.StringVar(&usernameFlag, "u", "root", "User for login (shorthand).")
	flag.StringVar(&passwordFlag, "password", "root", "Password to use when connecting to server.")
	flag.StringVar(&passwordFlag, "p", "root", "Password to use when connecting to server (shorthand).")
	flag.StringVar(&databaseFlag, "database", "", "Name of the database to use.")
	flag.StringVar(&databaseFlag, "d", "", "Name of the database to use (shorthand).")
}

func main() {
	flag.Parse() // Scan the arguments list

	if versionFlag {
		fmt.Println("Version:", APP_VERSION)
	} else {
		var data []*influxClient.Series;

		// Collect CPU data
		if u, err := cpus(prefixFlag); err == nil {
			data = append(data, u)
		} else {
			fmt.Fprintln(os.Stderr, err)
		}

		// Collect memory data
		if u, err := mem(prefixFlag); err == nil {
			data = append(data, u)
		} else {
			fmt.Fprintln(os.Stderr, err)
		}

		// Collect swap data
		if u, err := swap(prefixFlag); err == nil {
			data = append(data, u)
		} else {
			fmt.Fprintln(os.Stderr, err)
		}

		// Collect uptime data
		if u, err := uptime(prefixFlag); err == nil {
			data = append(data, u)
		} else {
			fmt.Fprintln(os.Stderr, err)
		}

		// Collect load average data
		if u, err := load(prefixFlag); err == nil {
			data = append(data, u)
		} else {
			fmt.Fprintln(os.Stderr, err)
		}

		// Collect network data
		if u, err := network(prefixFlag); err == nil {
			for _, v := range u {
				data = append(data, v)
			}
		} else {
			fmt.Fprintln(os.Stderr, err)
		}

		// Fill InfluxDB connection settings
		var config *influxClient.ClientConfig = nil;
		if databaseFlag != "" {
			config = new(influxClient.ClientConfig)

			config.Host = hostFlag
			config.Username = usernameFlag
			config.Password = passwordFlag
			config.Database = databaseFlag
		}

		send(config, data)
	}
}


/**
 * Interactions with InfluxDB
 */

func send(config *influxClient.ClientConfig, series []*influxClient.Series) error {
	// Pretty printer
	if config == nil || verboseFlag {
		for ks, serie := range series {
			nbCols := len(serie.Columns)

			fmt.Printf("\n#%d: %s\n", ks, serie.Name)

			for _, col := range serie.Columns {
				fmt.Printf("| %s\t", col)
			}
			fmt.Println("|")

			for _, value := range serie.Points {
				fmt.Print("| ")
				for i := 0; i < nbCols; i++ {
					fmt.Print(value[i], "\t| ")
				}
				fmt.Print("\n")
			}
		}
	}

	// Write to InfluxDB
	if config != nil {
		client, err := influxClient.NewClient(config)

		if err != nil {
			return err
		}

		client.WriteSeries(series)
	}

	return nil
}


/**
 * Gathering functions
 */

func cpus(prefix string) (*influxClient.Series, error) {
	if prefix != "" {
		prefix += "."
	}

	serie := &influxClient.Series{
		Name:    prefix + "cpu",
		Columns: []string{"id", "user", "nice", "sys", "idle", "wait", "total"},
		Points:  [][]interface{}{},
	}

	cpu := sigar.Cpu{}
	if err := cpu.Get(); err != nil {
		return nil, err
	}
	serie.Points = append(serie.Points, []interface{}{"cpu", cpu.User, cpu.Nice, cpu.Sys, cpu.Idle, cpu.Wait, cpu.Total()})

	cpus := sigar.CpuList{}
	cpus.Get()
	for i, cpu := range cpus.List {
		serie.Points = append(serie.Points, []interface{}{fmt.Sprint("cpu", i), cpu.User, cpu.Nice, cpu.Sys, cpu.Idle, cpu.Wait, cpu.Total()})
	}

	return serie, nil;
}

func mem(prefix string) (*influxClient.Series, error) {
	if prefix != "" {
		prefix += "."
	}

	serie := &influxClient.Series{
		Name:    prefix + "mem",
		Columns: []string{"free", "used", "actualfree", "actualused", "total"},
		Points:  [][]interface{}{},
	}

	mem := sigar.Mem{}
	if err := mem.Get(); err != nil {
		return nil, err
	}
	serie.Points = append(serie.Points, []interface{}{mem.Free, mem.Used, mem.ActualFree, mem.ActualUsed, mem.Total})

	return serie, nil
}

func swap(prefix string) (*influxClient.Series, error) {
	if prefix != "" {
		prefix += "."
	}

	serie := &influxClient.Series{
		Name:    prefix + "swap",
		Columns: []string{"free", "used", "total"},
		Points:  [][]interface{}{},
	}

	swap := sigar.Swap{}
	if err := swap.Get(); err != nil {
		return nil, err
	}
	serie.Points = append(serie.Points, []interface{}{swap.Free, swap.Used, swap.Total})

	return serie, nil
}

func uptime(prefix string) (*influxClient.Series, error) {
	if prefix != "" {
		prefix += "."
	}

	serie := &influxClient.Series{
		Name:    prefix + "uptime",
		Columns: []string{"length"},
		Points:  [][]interface{}{},
	}

	uptime := sigar.Uptime{}
	if err := uptime.Get(); err != nil {
		return nil, err
	}
	serie.Points = append(serie.Points, []interface{}{uptime.Length})

	return serie, nil
}

func load(prefix string) (*influxClient.Series, error) {
	if prefix != "" {
		prefix += "."
	}

	serie := &influxClient.Series{
		Name:    prefix + "load",
		Columns: []string{"one", "five", "fifteen"},
		Points:  [][]interface{}{},
	}

	load := sigar.LoadAverage{}
	if err := load.Get(); err != nil {
		return nil, err
	}
	serie.Points = append(serie.Points, []interface{}{load.One, load.Five, load.Fifteen})

	return serie, nil
}

func network(prefix string) ([]*influxClient.Series, error) {
	fi, err := os.Open("/proc/net/dev")
	if err != nil {
		return nil, err
	}
	defer fi.Close()

	var series []*influxClient.Series

	// Search interface
	scanner := bufio.NewScanner(fi)
	for scanner.Scan() {
		if u, err := network_parse_iface_line(prefix, scanner.Text()); err == nil && u != nil {
			series = append(series, u)
		}
	}

	return series, nil
}


/**
 * Network data related functions
 */

func network_parse_iface_line(prefix string, line string) (*influxClient.Series, error) {
	if prefix != "" {
		prefix += "."
	}

	tmp := strings.Split(line, ":")
	if len(tmp) < 2 {
		return nil, nil
	}

	iface := strings.Trim(tmp[0], " ")

	serie := &influxClient.Series{
		Name:    prefix + iface,
		Columns: []string{ "recv_bytes", "recv_packets", "recv_errs",
			           "recv_drop", "recv_fifo", "recv_frame",
			           "recv_compressed", "recv_multicast",
			           "trans_bytes", "trans_packets", "trans_errs",
			           "trans_drop", "trans_fifo", "trans_colls",
			           "trans_carrier", "trans_compressed" },
		Points:  [][]interface{}{},
	}

	tmp = strings.Fields(tmp[1])

	var points []interface{}

	for i := 0; i < len(serie.Columns); i++ {
		if v, err := strconv.Atoi(tmp[i]); err == nil {
			points = append(points, v)
		} else {
			points = append(points, 0)
		}
	}

	serie.Points = append(serie.Points, points)

	return serie, nil
}
