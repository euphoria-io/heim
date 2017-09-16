/*
github.com/twitter/snowflake in golang
*/

package gosnow

import (
    "testing"
    "github.com/stretchrcom/testify/assert"
)

func TestDefaultWorkId(t *testing.T) {
    id := DefaultWorkId()
    id2 := DefaultWorkId()
    t.Logf("id %v, next id %v", id, id2)

    if id != id2 {
        t.Errorf("different workd id, %v and  %v", id, id2)
    }
}

func TestNext(t *testing.T) {
    sf, err := Default()
    assert.Nil(t, err)

    id, err := sf.Next()
    assert.Nil(t, err)

    id2, err := sf.Next()
    assert.Nil(t, err)

    t.Logf("id %v, next id %v", id, id2)

    if id > id2 {
        t.Errorf("id %v is smaller then previous one %v", id2, id)
    }
}

func TestDuplicate(t *testing.T) {

    total := 1000 * 1000
    data := make(map[uint64]int)
    
    sf, err := Default()
    assert.Nil(t, err)

    var id, pre uint64
    for i := 0; i < total; i++ {

        id, err = sf.Next()
        assert.Nil(t, err)

        if id < pre {
            t.Errorf("id %v is samller than previous one %v", id, pre)
        }
        pre = id

        count := data[id]
        if count > 0 {
            t.Errorf("duplicate id %v %d", id, count)
        }
        data[id] = count + 1
    }

    length := len(data)
    t.Logf("map length %v", length)
    if length != total {
        t.Errorf("legth does not match want %v actual %d", total, length)
    }

}
