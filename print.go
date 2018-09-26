package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"syscall"
	"unsafe"
)

var winspool = syscall.NewLazyDLL("Winspool.drv")

type docInfo struct {
	pDocName    uintptr
	pOutputFile uintptr
	pDatatype   uintptr
}

type jobInfo struct {
	jobId        uint32
	pPrinterName uintptr
	pMachineName uintptr
	pUserName    uintptr
	pDocument    uintptr
	pDatatype    uintptr
	pStatus      uintptr
	status       uint32
	priority     uint32
	position     uint32
	totalPages   uint32
	pagesPrinted uint32
	submitted    systemTime
}

type systemTime struct {
	wYear         uint16
	wMonth        uint16
	wDayOfWeek    uint16
	wDay          uint16
	wHour         uint16
	wMinute       uint16
	wSecond       uint16
	wMilliseconds uint16
}

//typedef struct _JOB_INFO_1 {
//  DWORD      JobId;
//  LPTSTR     pPrinterName;
//  LPTSTR     pMachineName;
//  LPTSTR     pUserName;
//  LPTSTR     pDocument;
//  LPTSTR     pDatatype;
//  LPTSTR     pStatus;
//  DWORD      Status;
//  DWORD      Priority;
//  DWORD      Position;
//  DWORD      TotalPages;
//  DWORD      PagesPrinted;
//  SYSTEMTIME Submitted;
//}

//typedef struct _SYSTEMTIME {
//  WORD wYear;
//  WORD wMonth;
//  WORD wDayOfWeek;
//  WORD wDay;
//  WORD wHour;
//  WORD wMinute;
//  WORD wSecond;
//  WORD wMilliseconds;
//} SYSTEMTIME, *PSYSTEMTIME;

var jobStatuses = map[int]string{
	0x00000200: "Printer driver cannot print the job.",
	0x00001000: "Job has been delivered to the printer."}

type Printer struct {
	printer syscall.Handle
}

func NewPrinter(name string) (printer *Printer, err error) {
	printer = new(Printer)
	defer func() {
		if r := recover(); r != nil {
			err = errors.New(fmt.Sprint(r))
		}
	}()
	printer.open(name)
	// runtime.SetFinalizer(printer, func(p *Printer){
	// 	p.Close()
	// })
	return
}
func (p *Printer) GetDefaultPrinter(buf *uint16, bufN *uint32) (err error) {

	procGetDefaultPrinterW := winspool.NewProc("GetDefaultPrinterW")

	r1, _, msg := procGetDefaultPrinterW.Call(
		uintptr(unsafe.Pointer(buf)),
		uintptr(unsafe.Pointer(bufN)))
	if r1 != 1 {
		fmt.Println(msg)
	}
	fmt.Println(r1)
	return
}

func (p *Printer) PrintPostScriptFile(path string, title string) (jobId uint32, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = errors.New(fmt.Sprint(r))
		}
	}()
	jobId = p.openDoc(title)
	//p.openPage()
	defer p.closeDoc() //p.closePage()
	p.writeFile(path)
	return
}

func (p *Printer) GetJobStatus(jobId uint32) (uint32, error, string, uint32) {
	var getJob = winspool.NewProc("GetJobA") //GetJobW
	var level uint32 = 1
	//var bufSize uint32 = 10000
	var bufSize uint32 = 0
	var realSize uint32
	var job jobInfo

	ret, _, msg := getJob.Call(
		uintptr(unsafe.Pointer(p.printer)),
		uintptr(jobId),
		uintptr(level),
		uintptr(unsafe.Pointer(&job)),
		uintptr(bufSize),
		uintptr(unsafe.Pointer(&realSize)))
	bufSize = realSize

	var job2 jobInfo
	ret, _, msg = getJob.Call(
		uintptr(unsafe.Pointer(p.printer)),
		uintptr(jobId),
		uintptr(level),
		uintptr(unsafe.Pointer(&job2)),
		uintptr(bufSize),
		uintptr(unsafe.Pointer(&realSize)))

	return uint32(ret), msg, string(job2.pStatus), uint32(job2.status)
}

func (p *Printer) Close() {
	var closePrinter = winspool.NewProc("ClosePrinter")
	ret, _, msg := closePrinter.Call(uintptr(unsafe.Pointer(p.printer)))
	if ret != 1 {
		panic(msg)
	}
}

func (p *Printer) open(name string) {
	var openPrinter = winspool.NewProc("OpenPrinterW")
	ret, _, msg := openPrinter.Call(
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(name))),
		uintptr(unsafe.Pointer(&p.printer)),
		uintptr(unsafe.Pointer(nil)))
	if ret != 1 {
		panic(msg)
	}
}

// func (p *Printer) closePage() {
// 	var endDocPrinter = winspool.NewProc("EndDocPrinter");
// 	endDocPrinter.Call(uintptr(unsafe.Pointer(p.printer)))
// }

func (p *Printer) openDoc(name string) (jobId uint32) {
	var startDocPrinter = winspool.NewProc("StartDocPrinterW")
	var level uint32 = 1

	var doc docInfo
	doc.pDocName = uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(name)))
	doc.pOutputFile = uintptr(unsafe.Pointer(nil))
	doc.pDatatype = uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr("RAW")))

	ret, _, msg := startDocPrinter.Call(
		uintptr(unsafe.Pointer(p.printer)),
		uintptr(level),
		uintptr(unsafe.Pointer(&doc)))
	if ret == 0 {
		panic(msg)
	}
	jobId = uint32(ret)
	return jobId
}

func (p *Printer) closeDoc() {
	//var endPagePrinter = winspool.NewProc("EndPagePrinter")
	//endPagePrinter.Call(uintptr(unsafe.Pointer(p.printer)))
	var endDocPrinter = winspool.NewProc("EndDocPrinter")
	endDocPrinter.Call(uintptr(unsafe.Pointer(p.printer)))
}

// func (p *Printer) openPage() {
// 	var startPagePrinter = winspool.NewProc("StartPagePrinter")
// 	ret, _, msg := startPagePrinter.Call(uintptr(unsafe.Pointer(p.printer)))
// 	if ret != 1 {
// 		panic(msg)
// 	}
// }

func (p *Printer) writeFile(path string) {
	var writePrinter = winspool.NewProc("WritePrinter")
	document, err := ioutil.ReadFile(path)
	if nil != err {
		panic(err)
	}
	var bytesWritten uint32 = 0
	var docSize uint32 = uint32(len(document))
	ret, _, msg := writePrinter.Call(
		uintptr(unsafe.Pointer(p.printer)),
		uintptr(unsafe.Pointer(&document[2])),
		uintptr(docSize),
		uintptr(unsafe.Pointer(&bytesWritten)))
	fmt.Println(ret)
	if ret != 1 {
		panic(msg)
	}
}
