package mindvision

/*
#cgo linux,!android CFLAGS: -I../mvsdk/include
#cgo linux,!android LDFLAGS: -L${SRCDIR}/../mvsdk/lib -lMVSDK
#cgo darwin CFLAGS: -I../mvsdk/include
#cgo darwin LDFLAGS: -L${SRCDIR}/../mvsdk/lib -lmvsdk
#include "CameraDefine.h"
*/
import "C"
import (
	"fmt"
	"unsafe"
)

/*
CFLAGS： 指定头文件（.h文件）的路径,  -Idir	用于把新目录添加到include路径上，可以使用相对和绝对路径，“-I.”、“-I./include”、“-I/opt/include
LDFLAGS：gcc 等编译器会用到的一些优化参数，也可以在里面指定库文件的位置。-llibrary	链接时在标准搜索目录中寻找库文件，搜索名为liblibrary.a 或 liblibrary.so,-Ldir	用于把新目录添加到库搜索路径上，可以使用相对和绝对路径，“-L.”、“-L./include”、“-L/opt/include”
LIBS：告诉链接器要链接哪些库文件，如LIBS = -lpthread -liconv
-Wl选项告诉编译器将后面的参数传递给链接器
-rpath: 指定运行时库文件的目录

${SRCDIR} 是运行时目录 + 当前包名
*/

var (
	//CameraHandle=int(C.CameraHandle) 错误:这里是一些常量，C.CameraHandle是类型而不是常量
	DefautlPictureWidth  = 3088 //默认图像宽度
	DefautlPictureHeight = 2064 //默认图像高度
)

type Device struct {
	Id            int     `json:"id"`
	Sn            string  `json:"sn"`
	Series        string  `json:"series"`
	Name          string  `json:"name"`
	FriendlyName  string  `json:"friend_name"`
	LinkName      string  `json:"link_name"`
	SensorType    string  `json:"sensor_type"`
	PortType      string  `json:"port_type"`
	DriverVersion string  `json:"driver_version"`
	Expose        float32 `json:"expose"`
	Gain          int     `json:"gain"`
}

func (d *Device) ParseC(t C.tSdkCameraDevInfo) {
	d.Sn = C.GoString((*C.char)(unsafe.Pointer(&(t.acSn[0]))))
	d.Series = C.GoString((*C.char)(unsafe.Pointer(&(t.acProductSeries[0]))))
	d.Name = C.GoString((*C.char)(unsafe.Pointer(&(t.acProductName[0]))))
	d.FriendlyName = C.GoString((*C.char)(unsafe.Pointer(&(t.acFriendlyName[0]))))
	d.LinkName = C.GoString((*C.char)(unsafe.Pointer(&(t.acLinkName[0]))))
	d.SensorType = C.GoString((*C.char)(unsafe.Pointer(&(t.acSensorType[0]))))
	d.PortType = C.GoString((*C.char)(unsafe.Pointer(&(t.acPortType[0]))))
	d.DriverVersion = C.GoString((*C.char)(unsafe.Pointer(&(t.acDriverVersion[0]))))
}

//sdk错误处理
func sdkError(t C.int) (err error) {
	switch t {
	case C.CAMERA_STATUS_SUCCESS:
		err = nil
	case C.CAMERA_STATUS_FAILED:
		err = fmt.Errorf("操作失败")
	case C.CAMERA_STATUS_INTERNAL_ERROR:
		err = fmt.Errorf("内部错误")
	case C.CAMERA_STATUS_UNKNOW:
		err = fmt.Errorf("未知错误")
	case C.CAMERA_STATUS_NOT_SUPPORTED:
		err = fmt.Errorf("不支持该功能")
	case C.CAMERA_STATUS_NOT_INITIALIZED:
		err = fmt.Errorf("初始化未完成")
	case C.CAMERA_STATUS_PARAMETER_INVALID:
		err = fmt.Errorf("参数无效")
	case C.CAMERA_STATUS_PARAMETER_OUT_OF_BOUND:
		err = fmt.Errorf("参数越界")
	case C.CAMERA_STATUS_UNENABLED:
		err = fmt.Errorf("未使能")
	case C.CAMERA_STATUS_USER_CANCEL:
		err = fmt.Errorf("用户手动取消了，比如roi面板点击取消，返回")
	case C.CAMERA_STATUS_PATH_NOT_FOUND:
		err = fmt.Errorf("注册表中没有找到对应的路径")
	case C.CAMERA_STATUS_SIZE_DISMATCH:
		err = fmt.Errorf("获得图像数据长度和定义的尺寸不匹配")
	case C.CAMERA_STATUS_TIME_OUT:
		err = fmt.Errorf("超时错误")
	case C.CAMERA_STATUS_IO_ERROR:
		err = fmt.Errorf("硬件IO错误")
	case C.CAMERA_STATUS_COMM_ERROR:
		err = fmt.Errorf("通讯错误")
	case C.CAMERA_STATUS_BUS_ERROR:
		err = fmt.Errorf("总线错误")
	case C.CAMERA_STATUS_NO_DEVICE_FOUND:
		err = fmt.Errorf("没有发现设备")
	case C.CAMERA_STATUS_NO_LOGIC_DEVICE_FOUND:
		err = fmt.Errorf("未找到逻辑设备")
	case C.CAMERA_STATUS_DEVICE_IS_OPENED:
		err = fmt.Errorf("设备已经打开")
	case C.CAMERA_STATUS_DEVICE_IS_CLOSED:
		err = fmt.Errorf("设备已经关闭")
	case C.CAMERA_STATUS_DEVICE_VEDIO_CLOSED:
		err = fmt.Errorf("没有打开设备视频，调用录像相关的函数时，如果相机视频没有打开，则回返回该错误。")
	case C.CAMERA_STATUS_NO_MEMORY:
		err = fmt.Errorf("没有足够系统内存")
	case C.CAMERA_STATUS_FILE_CREATE_FAILED:
		err = fmt.Errorf("创建文件失败")
	case C.CAMERA_STATUS_FILE_INVALID:
		err = fmt.Errorf("文件格式无效")
	case C.CAMERA_STATUS_WRITE_PROTECTED:
		err = fmt.Errorf("写保护，不可写")
	case C.CAMERA_STATUS_GRAB_FAILED:
		err = fmt.Errorf("数据采集失败")
	case C.CAMERA_STATUS_LOST_DATA:
		err = fmt.Errorf("数据丢失，不完整")
	case C.CAMERA_STATUS_EOF_ERROR:
		err = fmt.Errorf("未接收到帧结束符")
	case C.CAMERA_STATUS_BUSY:
		err = fmt.Errorf("正忙(上一次操作还在进行中)，此次操作不能进行")
	case C.CAMERA_STATUS_WAIT:
		err = fmt.Errorf("需要等待(进行操作的条件不成立)，可以再次尝试trf")
	case C.CAMERA_STATUS_IN_PROCESS:
		err = fmt.Errorf("正在进行，已经被操作过")
	case C.CAMERA_STATUS_IIC_ERROR:
		err = fmt.Errorf("IIC传输错误")
	case C.CAMERA_STATUS_SPI_ERROR:
		err = fmt.Errorf("SPI传输错误")
	case C.CAMERA_STATUS_USB_CONTROL_ERROR:
		err = fmt.Errorf("USB控制传输错误")
	case C.CAMERA_STATUS_USB_BULK_ERROR:
		err = fmt.Errorf("USB BULK传输错误")
	case C.CAMERA_STATUS_SOCKET_INIT_ERROR:
		err = fmt.Errorf("网络传输套件初始化失败")
	case C.CAMERA_STATUS_GIGE_FILTER_INIT_ERROR:
		err = fmt.Errorf("网络相机内核过滤驱动初始化失败，请检查是否正确安装了驱动，或者重新安装。")
	case C.CAMERA_STATUS_NET_SEND_ERROR:
		err = fmt.Errorf("网络数据发送错误")
	case C.CAMERA_STATUS_DEVICE_LOST:
		err = fmt.Errorf("与网络相机失去连接，心跳检测超时")
	case C.CAMERA_STATUS_DATA_RECV_LESS:
		err = fmt.Errorf("接收到的字节数比请求的少")
	case C.CAMERA_STATUS_FUNCTION_LOAD_FAILED:
		err = fmt.Errorf("从文件中加载程序失败")
	case C.CAMERA_STATUS_CRITICAL_FILE_LOST:
		err = fmt.Errorf("程序运行所必须的文件丢失。")
	case C.CAMERA_STATUS_SENSOR_ID_DISMATCH:
		err = fmt.Errorf("固件和程序不匹配，原因是下载了错误的固件。")
	case C.CAMERA_STATUS_OUT_OF_RANGE:
		err = fmt.Errorf("参数超出有效范围。")
	case C.CAMERA_STATUS_REGISTRY_ERROR:
		err = fmt.Errorf("安装程序注册错误。请重新安装程序，或者运行安装目录Setup/Installer.exe")
	case C.CAMERA_STATUS_ACCESS_DENY:
		err = fmt.Errorf("禁止访问。指定相机已经被其他程序占用时，再申请访问该相机，会返回该状态。(一个相机不能被多个程序同时访问)")
	case C.CAMERA_STATUS_CAMERA_NEED_RESET:
		err = fmt.Errorf("表示相机需要复位后才能正常使用，此时请让相机断电重启，或者重启操作系统后，便可正常使用。")
	case C.CAMERA_STATUS_ISP_MOUDLE_NOT_INITIALIZED:
		err = fmt.Errorf("ISP模块未初始化")
	case C.CAMERA_STATUS_ISP_DATA_CRC_ERROR:
		err = fmt.Errorf("数据校验错误")
	case C.CAMERA_STATUS_MV_TEST_FAILED:
		err = fmt.Errorf("数据测试失败")
	case C.CAMERA_STATUS_INTERNAL_ERR1:
		err = fmt.Errorf("内部错误1")
	case C.CAMERA_STATUS_U3V_NO_CONTROL_EP:
		err = fmt.Errorf("U3V控制端点未找到")
	case C.CAMERA_STATUS_U3V_CONTROL_ERROR:
		err = fmt.Errorf("U3V控制通讯错误")
	default:
		err = fmt.Errorf("未知错误:%d", t)
	}
	return
}
