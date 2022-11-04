package cbytecache

import (
	"testing"
	"time"
)

type constHasher uint64

func (h constHasher) Sum64(_ string) uint64 {
	return uint64(h)
}

func TestCollision(t *testing.T) {
	conf := DefaultConfig(time.Minute, constHasher(1024))
	conf.CollisionCheck = true
	cache, err := New(conf)
	if err != nil {
		t.Fatal(err)
	}

	type stage struct {
		name, k0, k1 string
	}

	stages := []stage{
		{"collision0", "JIP4ndmjvUyTdJ2BbA", "GBmEU5yq7AyEAU3o20bz"},
		{"collision1", "aK6xTwbiOfgFvj3LASok", "OXzIyMtE1KJNN88v"},
		{"collision2", "2088g6ZKrnPP35ZOUq", "gavEav5xcvzoDBu3xIGCo"},
		{"collision3", "3meXEi3hD7ZCOz2cNNO", "0nea01UaDl74nQqgT04"},
		{"collision4", "tcz5yFctTrUDSTEPl0joCufM", "JiwCNPvZPXNWH1XiIxqQfVexTA"},
		{"collision5", "Vm2dVzZjPVMrLK3TPfyrvv", "6AlbmvC5sx3KZ8Qqjlv6lI6muGBgLl7"},
		{"collision6", "4SZiiYZVqETUObdz8Ddpo1R", "xs5qChLfJwvzdGN8gIh"},
		{"collision7", "DXRyexR8qYTXCaaFgD8CKo8773bf", "27Hio46dVY6ucklmD1dFZVzgHJ"},
		{"collision8", "NRL3yqtn20pgphWqfriIOFibm", "1AQNKtbcRbSNxvK8r"},
		{"collision9", "oOJcMciaGntvINFMb4ybGGiO08IX", "1MuKgoV1eFwaHlEnmoC"},
		{"collision10", "kARFVl2gqsH6z6fabnNu4", "wJPpDOmCIauXAsiIdTj"},
		{"collision11", "LXJOUWndGqbFJyjPGPsNALFL", "cdtvpEU3QeMggMyFLI04wyLjfYL"},
		{"collision12", "hW6PYvCFBXYpqvmdawNrtDW", "owKyVF5Xd6Ei7fq2b25nRuy0ftfDny"},
		{"collision13", "sJE7ul825IDIJiYv85yFOlEg2BKK", "eKkAbAvkw8BOLPAfIl8f0MktXFVt5Y"},
		{"collision14", "ZU6nXG0nBPXlShOx1", "8k8W2wXP1dKITneQLAqSb1cBff"},
		{"collision15", "jlVNv88eQsqKmwLFXETlkFjDBl81R0", "zgcMeLkkWUtN5KuvHQZ"},
		{"collision16", "KDsUGNFnLxLExhPC3TMSga", "qEiFRmFMkZMghcUu"},
		{"collision17", "6xlgAAI2AtzXI06cQqeq0rWrFDcHbZ", "vz4nsHxWaFma51oCncuRGUaOis"},
		{"collision18", "VlmtGRQyMHykCSgh1z", "up4WawvHUDk5fqZZ55O"},
		{"collision19", "rdMKfUpk3mCKNnbTmIm6Zv", "vyoe8x8GmdYaIEeJaQYDTeHez"},
		{"collision20", "MnQy7NqvfYZpUhJ8fRlHhIpobo", "STwoLGOiYG1szbz75K8vq"},
	}

	testC7n := func(t *testing.T, cache *Cache, st *stage) {
		_ = cache.Set(st.k0, dataPool[0])
		err := cache.Set(st.k1, dataPool[1])
		if err != ErrEntryCollision {
			t.Error(st.name, "collision miss, expect", ErrEntryCollision, "got", err)
		}
	}

	for _, st := range stages {
		t.Run(st.name, func(t *testing.T) { testC7n(t, cache, &st) })
	}
}
