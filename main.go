package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

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

var FileName string

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
	//fmt.Println("请输入要使用的设备索引:")
	//fmt.Scanf("%d", &selectIdx)
	if err := c.ActiveCamera(selectIdx); err != nil {
		panic(err)
	}

	for {
		fmt.Println("请输入图片名称:")
		fmt.Scanf("%s", &FileName)
		grap(c, selectIdx)
	}

	/*
		fmt.Println("模式： 1 预览， 2 抓图")
		runMode := 0
		fmt.Scanf("%d", &runMode)
		switch runMode {
		case 1:
			preview(c, selectIdx)
		case 2:
			grap(c, selectIdx)
		default:
			preview(c, selectIdx)
		}
	*/

}

func preview(c *mindvision.Camera, selectIdx int) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	f, err := os.OpenFile("temp.stream", os.O_CREATE|os.O_RDWR|os.O_APPEND, 0777)
	if err != nil {
		fmt.Println(err)
	}

	if err := c.Preview(ctx, 500, 900, f); err != nil {
		panic(err)
	}

}

func grap(c *mindvision.Camera, selectIdx int) {

	if err := c.SetupForGrab(200, 1); err != nil {
		panic(err)
	}

	c.Grab(fmt.Sprintf("test_%s.bmp", FileName))

	/*
			for i := 0; i < 10; i++ {
				 if err := c.Grab(fmt.Sprintf("test_%d.bmp", i)); err != nil {
		   			panic(err)
				}
			}
	*/
}
