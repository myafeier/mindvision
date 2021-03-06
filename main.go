package main

import (
	"encoding/json"
	"fmt"

	"github.com/myafeier/mindvision/mindvision"
)

func init() {
}

/*
C语言类型	CGO类型	Go语言类型
char	C.char	byte
singed char	C.schar	int8
unsigned char	C.uchar	uint8
short	C.short	int16
unsigned short	C.ushort	uint16
int	C.int	int32
unsigned int	C.uint	uint32
long	C.long	int32
unsigned long	C.ulong	uint32
long long int	C.longlong	int64
unsigned long long int	C.ulonglong	uint64
float	C.float	float32
double	C.double	float64
size_t	C.size_t	uint
*/

/**
1、go build -x 会显示编译细节，很好的调试工具
2、假设C中有个 int *p; Go中赋值: C.p= (*C.int)(unsafe.Pointer(new(int)))
*/

func main() {
	c := new(mindvision.Camera)
	c.Init("")
	if list, err := c.EnumerateDevice(); err != nil {
		panic(err)
	} else {
		fmt.Printf("发现:%d套设备\n", len(list))
		for k, v := range list {
			fmt.Printf("k: %d, v: %+v\n", k, v)
		}
		if js, err := json.Marshal(list); err != nil {
			fmt.Printf("marshal error: %s\n", err.Error())
		} else {
			fmt.Printf("JSON: %s\n", js)
		}
	}
	selectIdx := 0
	fmt.Println("请输入要使用的设备索引:")
	fmt.Scanf("%d", &selectIdx)
	if err := c.ActiveCamera(selectIdx, 2); err != nil {
		panic(err)
	}
	for i := 0; i < 10; i++ {
		if f, err := c.Grab(); err != nil {
			panic(err)
		} else {
			fmt.Println(f)
		}
	}
}
