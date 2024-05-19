package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/miekg/dns"
)

var records = map[string]string{
	"dev.service.tw.":  "192.168.55.21",
	"test.service.tw.": "192.168.55.21",
	"prd.service.tw.":  "192.168.55.21",
	"dev.egg.local.":   "192.168.55.21",
}

const upstreamDNS = "8.8.8.8:53" // Example upstream DNS server (Google Public DNS)

func parseQuery(m *dns.Msg, w dns.ResponseWriter) bool {
	found := false
	for _, q := range m.Question {
		switch q.Qtype {
		case dns.TypeA:
			log.Printf("Query for %s\n", q.Name)
			ip := records[strings.ToLower(q.Name)]
			if ip != "" {
				rr, err := dns.NewRR(fmt.Sprintf("%s A %s", q.Name, ip))
				if err != nil {
					log.Printf("Failed to create DNS RR: %s\n", err)
				} else {
					m.Answer = append(m.Answer, rr)
					found = true
				}
			} else {
				log.Printf("No record found for %s\n", q.Name)
			}
		default:
			log.Printf("Unsupported query type: %d\n", q.Qtype)
		}
	}
	return found
}

func forwardQuery(r *dns.Msg, w dns.ResponseWriter) {
	c := new(dns.Client)
	resp, _, err := c.Exchange(r, upstreamDNS)
	if err != nil {
		log.Printf("Failed to forward query: %s\n", err)
		dns.HandleFailed(w, r)
		return
	}
	w.WriteMsg(resp)
}

func handleDnsRequest(w dns.ResponseWriter, r *dns.Msg) {
	log.Printf("Received DNS request for %v\n", r.Question)
	m := new(dns.Msg)
	m.SetReply(r)
	m.Compress = false

	if r.Opcode == dns.OpcodeQuery {
		if !parseQuery(m, w) {
			forwardQuery(r, w)
			return
		}
	} else {
		log.Printf("Unsupported DNS Opcode: %d\n", r.Opcode)
	}

	w.WriteMsg(m)
}

func main() {
	// attach request handler func
	dns.HandleFunc(".", handleDnsRequest)

	// start server
	port := 53
	server := &dns.Server{Addr: ":" + strconv.Itoa(port), Net: "udp"}
	log.Printf("Starting server on sport %d\n", port)

	// Signal handling for graceful shutdown
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sig
		log.Println("Shutting down server...")
		if err := server.Shutdown(); err != nil {
			log.Fatalf("Failed to shutdown server: %s\n", err)
		}
	}()

	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Failed to start server: %s\n", err)
	}
}
