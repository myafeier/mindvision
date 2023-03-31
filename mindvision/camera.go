package mindvision

/*
#cgo linux,!android CFLAGS: -I../mvsdk/include
#cgo linux,!android LDFLAGS: -L../mvsdk/lib -lMVSDK
#cgo darwin CFLAGS: -I../mvsdk/include
#cgo darwin LDFLAGS: -L${SRCDIR}/../mvsdk/lib -lmvsdk
#include "CameraApi.h"
#include <stdio.h>
CameraHandle handle;
*/
import "C"
import (
	"context"
	"fmt"
	"image"
	"image/png"
	"io"
	"log"
	"unsafe"

	"github.com/pkg/errors"
	//"github.com/myafeier/log"
)

func init() {
	log.SetPrefix("[GoMindVersion]")
	log.SetFlags(log.Llongfile | log.Ltime)
}

type Camera struct {
	devices  [32]C.tSdkCameraDevInfo
	idx      int     //设备序号
	bufsize  int     //抓图缓存大小
	width    int     //图片最大尺寸
	height   int     //图片最大尺寸
	expose   float64 //曝光时间
	gain     int     //增益
	filepath string
}

func (s *Camera) Init(filepath string) (err error) {
	status := C.CameraSdkInit(C.int(0))
	err = sdkError(status)
	if err != nil {
		return
	}

	if filepath == "" {
		s.filepath = "./"
	} else {
		s.filepath = filepath
	}
	return
}

func (s *Camera) UnInit() {
	C.CameraUnInit(C.handle)
}

// 查看设备列表
func (s *Camera) EnumerateDevice() (list []*Device, err error) {
	var count int = 32
	// CameraEnumerateDevice 要求传入数组指针，及数组长度指针
	status := C.CameraEnumerateDevice((*C.tSdkCameraDevInfo)(unsafe.Pointer(&(s.devices[0]))), (*C.int)(unsafe.Pointer(&count)))
	err = sdkError(status)
	if err != nil {
		log.Println(err.Error())
		return
	}
	for i := 0; i < count; i++ {
		t := new(Device)
		t.ParseC(s.devices[i])
		t.Id = i + 1
		list = append(list, t)
	}

	return
}

// 选择并激活相机
func (s *Camera) ActiveCamera(idx int) (err error) {
	status := C.CameraInit((*C.tSdkCameraDevInfo)(unsafe.Pointer(&(s.devices[idx]))), C.int(-1), C.int(-1), (*C.CameraHandle)(unsafe.Pointer(&C.handle)))
	err = sdkError(status)
	if err != nil {
		log.Println(err.Error())
		return
	}

	// 获取相机的参数
	var capability C.tSdkCameraCapbility

	status = C.CameraGetCapability(C.handle, (*C.tSdkCameraCapbility)(unsafe.Pointer(&capability)))
	err = sdkError(status)
	if err != nil {
		log.Println(err.Error())
		return
	}
	if int(capability.sIspCapacity.bMonoSensor) == 1 {
		s.bufsize = int(capability.sResolutionRange.iWidthMax * capability.sResolutionRange.iHeightMax)
		status = C.CameraSetIspOutFormat(C.handle, C.CAMERA_MEDIA_TYPE_MONO8)

	} else {
		s.bufsize = int(capability.sResolutionRange.iWidthMax*capability.sResolutionRange.iHeightMax) * 3
		status = C.CameraSetIspOutFormat(C.handle, C.CAMERA_MEDIA_TYPE_BGR8)
	}
	s.width = int(capability.sResolutionRange.iWidthMax)
	s.height = int(capability.sResolutionRange.iHeightMax)

	err = sdkError(status)
	if err != nil {
		log.Println(err.Error())
		return
	}
	return
}

// 预览
//
//	exposureTime 曝光时间，单位：毫秒
//	gain int 增益
//	previewChan 图片流
//	ctx 停止信号
func (s *Camera) Preview(ctx context.Context, exposureTime int, gain int, w io.Writer) (err error) {

	// 相机模式切换成连续采集, 0为连续采集，1位软触发采集，用户每次调用CameraSoftTrigger(hCamera)获取一张图片
	status := C.CameraSetTriggerMode(C.handle, C.int(0))
	err = sdkError(status)
	if err != nil {
		err = errors.WithStack(err)
		return
	}

	status = C.CameraSetAnalogGain(C.handle, C.int(gain))
	err = sdkError(status)
	if err != nil {
		err = errors.WithStack(err)
		return
	}

	// 手动曝光模式,并设置曝光时间
	status = C.CameraSetAeState(C.handle, C.int(0))
	err = sdkError(status)
	if err != nil {
		err = errors.WithStack(err)
		return
	}

	//曝光0.5 s
	status = C.CameraSetExposureTime(C.handle, C.double(exposureTime*1000))
	err = sdkError(status)
	if err != nil {
		err = errors.WithStack(err)
		return
	}

	// 设置预览分辨率
	var roiResolution C.tSdkImageResolution
	roiResolution.iIndex = 0xff
	roiResolution.iWidth = C.int(s.width / 2)
	roiResolution.iHeight = C.int(s.height / 2)
	roiResolution.iWidthFOV = C.int(s.width / 2)
	roiResolution.iHeightFOV = C.int(s.height / 2)

	//roiResolution.iWidthZoomSw = 4
	//roiResolution.iHeightZoomSw = 4

	status = C.CameraSetImageResolution(C.handle, (*C.tSdkImageResolution)(unsafe.Pointer(&roiResolution)))
	err = sdkError(status)
	if err != nil {
		err = errors.WithStack(err)
		return
	}

	status = C.CameraSetRotate(C.handle, 2)
	err = sdkError(status)
	if err != nil {
		err = errors.WithStack(err)
		return
	}
	//让SDK进入工作模式
	status = C.CameraPlay(C.handle)
	err = sdkError(status)
	if err != nil {
		err = errors.WithStack(err)
		return
	}

	t := C.int(0)
	rawDataPtr := (**C.BYTE)(unsafe.Pointer(&t)) //这里是指向指针的指针，所以用一个int存储即可
	log.Printf("rawptr init:%+v\n", rawDataPtr)

	var exit bool
	go func() {
		<-ctx.Done()
		exit = true
	}()

	for {
		if exit {
			log.Printf("preview exit")
			break
		}

		var frameInfo C.tSdkFrameHead
		//rawDataPtr := C.CameraAlignMalloc(C.int(s.bufsize), 4)
		//status := C.CameraGetImageBuffer(C.handle, (*C.tSdkFrameHead)(unsafe.Pointer(&frameInfo)), (**C.BYTE)(unsafe.Pointer(&rawDataPtr)), 6000)
		status = C.CameraGetImageBuffer(C.handle, (*C.tSdkFrameHead)(unsafe.Pointer(&frameInfo)), rawDataPtr, 6000)
		err = sdkError(status)
		if err != nil {
			err = errors.WithStack(err)
			return
		}

		img := image.NewGray(image.Rect(0, 0, int(frameInfo.iWidth), int(frameInfo.iHeight)))
		img.Pix = C.GoBytes(unsafe.Pointer(*rawDataPtr), C.int(s.bufsize))
		err = png.Encode(w, img)
		if err != nil {
			err = errors.WithStack(err)
			return
		}

		status = C.CameraReleaseImageBuffer(C.handle, *rawDataPtr)
		err = sdkError(status)
		if err != nil {
			err = errors.WithStack(err)
			return
		}

	}

	return
}

// 设置以抓图
func (s *Camera) SetupForGrab(gain int, exposeSecond float32) (err error) {
	// 相机模式切换成连续采集, 0为连续采集，1位软触发采集，用户每次调用CameraSoftTrigger(hCamera)获取一张图片
	status := C.CameraSetTriggerMode(C.handle, C.int(1))
	err = sdkError(status)
	if err != nil {
		err = errors.WithStack(err)
		return
	}

	// 手动曝光模式,并设置曝光时间
	status = C.CameraSetAeState(C.handle, C.int(0))
	err = sdkError(status)
	if err != nil {
		err = errors.WithStack(err)
		return
	}
	//曝光3秒
	status = C.CameraSetExposureTime(C.handle, C.double(exposeSecond*1000000))
	err = sdkError(status)
	if err != nil {
		err = errors.WithStack(err)
		return
	}
	//设定增益
	status = C.CameraSetAnalogGain(C.handle, C.int(gain))
	err = sdkError(status)
	if err != nil {
		err = errors.WithStack(err)
		return
	}
	//旋转90度
	C.CameraSetRotate(C.handle, 2)

	//让SDK进入工作模式
	status = C.CameraPlay(C.handle)
	err = sdkError(status)
	if err != nil {
		err = errors.WithStack(err)
		return
	}
	log.Println("设备已就绪，等待指令")
	return
}

// 获取一张图片
func (s *Camera) Grab(fn string) (err error) {
	// 分配RGB buffer，用来存放ISP输出的图像
	//备注：从相机传输到PC端的是RAW数据，在PC端通过软件ISP转为RGB数据（如果是黑白相机就不需要转换格式，但是ISP还有其它处理，所以也需要分配这个buffer）

	//log.Printf("bufsize: %d\n", s.bufsize)
	outputPtr := C.CameraAlignMalloc(C.int(s.bufsize), 16)
	log.Printf("outptr init:%+v\n", outputPtr)
	defer func() {
		C.CameraAlignFree(outputPtr)
		log.Printf("outptr defer:%+v\n", outputPtr)
	}()

	//当关闭连续取图时，软触发取图
	C.CameraSoftTrigger(C.handle)

	var frameInfo C.tSdkFrameHead
	//rawDataPtr := C.CameraAlignMalloc(C.int(s.bufsize), 4)
	t := C.int(0)
	rawDataPtr := (**C.BYTE)(unsafe.Pointer(&t))
	log.Printf("rawptr init:%+v\n", rawDataPtr)
	//status := C.CameraGetImageBuffer(C.handle, (*C.tSdkFrameHead)(unsafe.Pointer(&frameInfo)), (**C.BYTE)(unsafe.Pointer(&rawDataPtr)), 6000)
	status := C.CameraGetImageBuffer(C.handle, (*C.tSdkFrameHead)(unsafe.Pointer(&frameInfo)), rawDataPtr, 6000)
	err = sdkError(status)
	if err != nil {
		err = errors.WithStack(err)
		return
	}
	log.Printf("rawptr after get:%+v\n", rawDataPtr)
	status = C.CameraImageProcess(C.handle, *rawDataPtr, outputPtr, (*C.tSdkFrameHead)(unsafe.Pointer(&frameInfo)))
	err = sdkError(status)
	if err != nil {
		err = errors.WithStack(err)
		return
	}
	log.Printf("outptr after process:%+v\n", outputPtr)
	//	blob := C.GoBytes(unsafe.Pointer(outputPtr), C.int(s.bufsize))

	//fmt.Printf("head: %v\n,blob:%v\n", frameInfo, blob)
	status = C.CameraReleaseImageBuffer(C.handle, *rawDataPtr)
	err = sdkError(status)
	if err != nil {
		err = errors.WithStack(err)
		return
	}
	log.Printf("rawptr after release:%+v\n", rawDataPtr)
	fn = fmt.Sprintf(s.filepath + fn)
	status = C.CameraSaveImage(C.handle, C.CString(fn), outputPtr, (*C.tSdkFrameHead)(unsafe.Pointer(&frameInfo)), C.FILE_BMP, 0)
	err = sdkError(status)
	if err != nil {
		err = errors.WithStack(err)
		return
	}
	return
}

// 设定增益
func (s *Camera) SetGain(gain int) (err error) {

	status := C.CameraSetAnalogGain(C.handle, C.int(gain))
	err = sdkError(status)
	if err != nil {
		log.Println(err.Error())
		return
	}
	return
}

// 设定曝光时间
func (s *Camera) SetExpose(expose float32) (err error) {
	status := C.CameraSetExposureTime(C.handle, C.double(expose*1000000))
	err = sdkError(status)
	if err != nil {
		log.Println(err.Error())
		return
	}
	return
}
