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
	"image"
	"image/jpeg"
	"io"
	"log"
	"net/http"
	"sync"
	"unsafe"

	"github.com/pkg/errors"
	//"github.com/myafeier/log"
)

var WithoutHardware bool = false //是否脱机测试
type CameraMode uint8            //摄像头工作模式

const (
	CameraModeOfPreview = 1 //预览模式，连续抓取
	CameraModeOfCaputre = 2 //照片模式,默认
)

func init() {
	log.SetPrefix("[GoMindVersion]")
	log.SetFlags(log.Llongfile | log.Ltime)
}

type Camera struct {
	devices     [32]C.tSdkCameraDevInfo
	idx         int     //设备序号
	bufsize     int     //抓图缓存大小
	width       int     //图片最大尺寸
	height      int     //图片最大尺寸
	expose      float64 //曝光时间
	gain        int     //增益
	filepath    string
	mode        CameraMode
	stopPreview bool          //停止预览
	mjpegOption *jpeg.Options //
}

func (s *Camera) Init(filepath string, mjpegOption *jpeg.Options) (err error) {
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
	s.mjpegOption = mjpegOption
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
	if WithoutHardware {
		return
	}
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
	status = C.CameraSetIspOutFormat(C.handle, C.CAMERA_MEDIA_TYPE_MONO8)
	err = sdkError(status)
	if err != nil {
		log.Println(err.Error())
		return
	}

	s.width = int(capability.sResolutionRange.iWidthMax)
	s.height = int(capability.sResolutionRange.iHeightMax)

	// 直接输出为8位灰度图片
	s.bufsize = s.width * s.height

	/*
		if int(capability.sIspCapacity.bMonoSensor) == 1 {
			s.bufsize = int(capability.sResolutionRange.iWidthMax * capability.sResolutionRange.iHeightMax)
			status = C.CameraSetIspOutFormat(C.handle, C.CAMERA_MEDIA_TYPE_MONO8)

		} else {
			s.bufsize = int(capability.sResolutionRange.iWidthMax*capability.sResolutionRange.iHeightMax) * 3
			status = C.CameraSetIspOutFormat(C.handle, C.CAMERA_MEDIA_TYPE_BGR8)
		}
	*/

	return
}

var mutex sync.Mutex

// 切换工作模式，并进入工作状态
func (s *Camera) ChangeMode(mode CameraMode, exposeTime float32, gain int) (err error) {
	mutex.Lock()
	defer mutex.Unlock()
	if mode == s.mode {
		return
	}

	if mode == CameraModeOfCaputre {
		err = s.setupForCapture(gain, exposeTime)

	} else if mode == CameraModeOfPreview {
		err = s.setupForPreview(s.width, s.height)
	}
	return
}

// 设置以预览
func (s *Camera) setupForPreview(width, height int) (err error) {
	// 相机模式切换成连续采集, 0为连续采集，1位软触发采集，用户每次调用CameraSoftTrigger(hCamera)获取一张图片
	status := C.CameraSetTriggerMode(C.handle, C.int(0))
	err = sdkError(status)
	if err != nil {
		err = errors.WithStack(err)
		return
	}
	// 自动曝光模式
	status = C.CameraSetAeState(C.handle, C.int(0))
	err = sdkError(status)
	if err != nil {
		err = errors.WithStack(err)
		return
	}
	status = C.CameraSetExposureTime(C.handle, C.double(30*1000))
	err = sdkError(status)
	if err != nil {
		err = errors.WithStack(err)
		return
	}
	/*
		// 设置预览分辨率
		var roiResolution C.tSdkImageResolution
		roiResolution.iIndex = 0xff
		roiResolution.iWidth = C.int(width)
		roiResolution.iHeight = C.int(height)
		roiResolution.iWidthFOV = C.int(width)
		roiResolution.iHeightFOV = C.int(height)
		log.Printf("%+v \n", roiResolution)

		status = C.CameraSetImageResolution(C.handle, (*C.tSdkImageResolution)(unsafe.Pointer(&roiResolution)))
		err = sdkError(status)
		if err != nil {
			err = errors.WithStack(err)
			return
		}
	*/

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
	s.stopPreview = false
	s.mode = CameraModeOfPreview
	log.Println("设备进入预览模式")
	return
}

// 设置以照相
func (s *Camera) setupForCapture(gain int, exposeSecond float32) (err error) {
	// 相机模式切换成连续采集, 0为连续采集，1位软触发采集，用户每次调用CameraSoftTrigger(hCamera)获取一张图片
	status := C.CameraSetTriggerMode(C.handle, C.int(1))
	err = sdkError(status)
	if err != nil {
		err = errors.WithStack(err)
		return
	}

	/*

		// 设置预览分辨率
		var roiResolution C.tSdkImageResolution
		roiResolution.iIndex = 0xff
		roiResolution.iWidth = C.int(s.width)
		roiResolution.iHeight = C.int(s.height)
		roiResolution.iWidthFOV = C.int(s.width)
		roiResolution.iHeightFOV = C.int(s.height)
		log.Printf("%+v \n", roiResolution)

		status = C.CameraSetImageResolution(C.handle, (*C.tSdkImageResolution)(unsafe.Pointer(&roiResolution)))
		err = sdkError(status)
		if err != nil {
			err = errors.WithStack(err)
			return
		}
	*/

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
	s.stopPreview = true
	s.mode = CameraModeOfCaputre
	log.Println("设备进入拍照模式")
	return
}

// 停止预览
func (s *Camera) StopPreview() {
	s.stopPreview = true
}

// 获取mjpeg视频流
func (s *Camera) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "multipart/x-mixed-replace; boundary=frame")
	boundary := "\r\n--frame\r\nContent-Type: image/jpeg\r\n\r\n"

	t := C.int(0)
	rawDataPtr := (**C.BYTE)(unsafe.Pointer(&t)) //这里是指向指针的指针，所以用一个int存储即可
	log.Printf("rawptr init:%+v\n", rawDataPtr)

	outputPtr := C.CameraAlignMalloc(C.int(s.bufsize), 16)
	log.Printf("outptr init:%+v\n", outputPtr)
	defer func() {
		C.CameraAlignFree(outputPtr)
		log.Printf("outptr defer:%+v\n", outputPtr)
	}()

	s.stopPreview = false
	i := 0

	for {
		if s.stopPreview {
			log.Println("preview mode closed")
			return
		}
		i += 1
		log.Print(i)

		var frameInfo C.tSdkFrameHead
		status := C.CameraGetImageBuffer(C.handle, (*C.tSdkFrameHead)(unsafe.Pointer(&frameInfo)), rawDataPtr, 6000)
		err := sdkError(status)
		if err != nil {
			log.Println(err)
			return
		}

		status = C.CameraImageProcess(C.handle, *rawDataPtr, outputPtr, (*C.tSdkFrameHead)(unsafe.Pointer(&frameInfo)))
		err = sdkError(status)
		if err != nil {
			err = errors.WithStack(err)
			return
		}
		status = C.CameraReleaseImageBuffer(C.handle, *rawDataPtr)
		err = sdkError(status)
		if err != nil {
			log.Println(err)
			return
		}

		_, err = io.WriteString(w, boundary)
		if err != nil {
			log.Println(err)
			return
		}

		data := C.GoBytes(unsafe.Pointer(outputPtr), C.int(s.bufsize))
		img := image.NewGray(image.Rect(0, 0, int(frameInfo.iWidth), int(frameInfo.iHeight)))
		copy(img.Pix, data)

		/*
			f1, _ := os.OpenFile(fmt.Sprintf("t%d.data", i), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
			f1.Write(data)
			f1.Close()

					f, _ := os.OpenFile(fmt.Sprintf("t%d.jpg", i), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
			jpeg.Encode(f, img, s.mjpegOption)
			f.Close()
		*/

		err = jpeg.Encode(w, img, s.mjpegOption)
		if err != nil {
			log.Println(err)
			return
		}
		_, err = io.WriteString(w, "\r\n")
		if err != nil {
			log.Println(err)
			return
		}

	}

	return
}

// 获取一张图片
func (s *Camera) Grab(fn string) (err error) {
	if WithoutHardware {
		return
	}
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

	//img:=image.NewGray(image.Rect(0,0,int(frameInfo.iWidth),int(frameInfo.iHeight)))
	// 可以通过循环读取rawDataPtr数据插入到img中

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
	if WithoutHardware {
		return
	}
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
	if WithoutHardware {
		return
	}
	status := C.CameraSetExposureTime(C.handle, C.double(expose*1000000))
	err = sdkError(status)
	if err != nil {
		log.Println(err.Error())
		return
	}
	return
}
