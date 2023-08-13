## GO client for EZTV API
This package provides a simple API client for to the public `EZTV` [API](https://eztv.re/api/).

---
## Download
```shell
go get github.com/PauliusLozys/eztv
```
---
## Example usage
```go
package main

import (
	"context"
	"fmt"

	"github.com/PauliusLozys/eztv"
)

func main() {
	client := eztv.New() // Create new client
	page, err := client.GetTorrents(context.TODO(), eztv.URLOptions{
		Limit:  10,
		Page:   1,
		ImdbID: "0944947",
	})
	if err != nil {
		panic(err)
	}
	for _, t := range page.Torrents {
		fmt.Println(t.Title)
	}
}
```
 ## Example torrent stream usage
```go
package main

import (
	"context"
	"fmt"

	"github.com/PauliusLozys/eztv"
)

func main() {
	client := eztv.New()
	for s := range client.TorrentStream(context.Background(), eztv.StreamOptions{LastTorrentID: 0, ImdbID: "tt7908628"}) {
		if s.Err != nil {
			fmt.Println("Error:", s.Err)
			continue
		}
		fmt.Println(s.Torrent.ID, s.Torrent.Title)
	}
}
```