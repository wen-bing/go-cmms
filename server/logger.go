package main

import (
	"fmt"
	"log"
	"time"
)

func logger(start time.Time, from string, in string, out string) {

	s1 := fmt.Sprintf(`%-20s %10s`, from, time.Since(start))
	log.Printf(`%-35s » %-50s « %s`, s1, in, out)
}

// Site.UserList             -> 1.364043ms     » Channel 1, User 45 testw1 Worker         « 2 Sites
// Site.Get                  ->   1.151579ms » Site 4                                   « Edinburgh - SMiC
// Site.MachineList          ->   2.986588ms » Channel 1, Site 4, User 45 testw1 Worker « 3 machines
// Site.UserList             ->   1.354785ms » Channel 1, User 45 testw1 Worker         « 2 Sites
// Site.Get                  ->   1.109204ms » Site 2                                   « Edinburgh - Factory
// Site.MachineList          ->   5.597996ms » Channel 1, Site 2, User 45 testw1 Worker « 9 machines
// Site.UserList             ->   1.164497ms » Channel 1, User 45 testw1 Worker         « 2 Sites
