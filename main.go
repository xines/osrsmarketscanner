/*
	Copyright © 2021, Xines
	All rights reserved.

	This source code is licensed under the BSD-style license found in the
	LICENSE file in the root directory of this source tree.
*/

package main

import (
	"osrsmarketscanner/discord"
	"time"
)

func main() {
	discord.StartDiscordBot("Starting Bot.", "NO-ID")
	time.Sleep(3 * time.Second)
}
