package onec

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"log"
	"math"
	"os"
	"regexp"
	"strconv"
	"strings"
)

var Ver8380 = [4]byte{8, 3, 8, 0}

var (
	TableDescriptionPattern = regexp.MustCompile(`\{"(\S+)".*\n\{"Fields",\n([\s\S]*)\n\},\n\{"Indexes"(?:,|)([\s\S]*)\},\n\{"Recordlock","(\d)+"\},\n\{"Files",(\S+)\}\n\}`)
	FieldDescriptionPattern = regexp.MustCompile(`\{"(\w+)","(\w+)",(\d+),(\d+),(\d+),"(\w+)"\}(?:,|)`)
)

const RootObjectOffset uint64 = 2
const BlobChunkSize uint32 = 256

type headDB struct { //8s4bIiI
	Cd            [8]byte //  “1CDBMSV8”
	Ver           [4]byte //8 3 8 0
	NumberOfPages int32   //number of pages
	Unknown       uint32  // unknown
	PageSize      uint32  // page size
}

type BaseOnec struct {
	db               *os.File
	HeadDB           headDB
	TableDescription []Table
}

type Table struct {
	Name        string
	RecordLock  bool
	DataOffset  int
	BlobOffset  int
	IndexOffset int
	RowLength   int
	Fields      map[string]Field
}

func ReadBytes(db *os.File, position uint64, lenth uint32) []byte {
	_, err := db.Seek(int64(position), 0)
	if err != nil {
		log.Fatal(err)
	}
	bytes := make([]byte, lenth)
	_, err = db.Read(bytes)
	if err != nil {
		log.Fatal(err)
	}
	return bytes
}

func readHeadDB(db *os.File) (headDB, error) {
	headDB := headDB{}
	/* do not work on benchmark
	rd := bufio.NewReader(db)
	buf, err := rd.Peek(24)
	if err != nil {
		log.Fatal("rd.Read failed", err)
		return headDB, err
	}
	fmt.Println(buf) // - takes on different values on benchmark
	*/
	buf := ReadBytes(db, 0, 24)
	buffer := bytes.NewBuffer(buf)
	err := binary.Read(buffer, binary.LittleEndian, &headDB)
	if err != nil {
		log.Fatal("binary.Read failed", err)
		return headDB, err
	}

	return headDB, nil
}

func readBlobStream(db *os.File, offset uint64, pageSize uint32, dataPagesOffsets []uint32) []byte {
	nextBlock := uint32(1)
	data := make([]byte, 0, 250)
	currentOffset := offset
	for nextBlock != 0 {
		b := ReadBytes(db, currentOffset, BlobChunkSize)
		nextBlock = binary.LittleEndian.Uint32(b[:4])
		nextPage := uint32(nextBlock) * BlobChunkSize / pageSize
		currentOffset = uint64(dataPagesOffsets[nextPage])*uint64(pageSize) + uint64(uint32(nextBlock)*BlobChunkSize%pageSize)
		size := binary.LittleEndian.Uint16(b[4:6])
		data = append(data, b[6:6+size]...)
	}

	return data
}

func readBlockOfReplacemant(BO *BaseOnec, blobOffsetSlice []uint32) []uint32 {
	var blobChunkOffset uint32 = 1
	pageSize := BO.HeadDB.PageSize

	blobOffset := blobOffsetSlice[0]
	//if blobSize == 0 {return make([]uint32, 0)}
	//fmt.Println(int64(blobOffset)*int64(pageSize) + int64(blobChunkOffset*BlobChunkSize))
	b := readBlobStream(BO.db, uint64(blobOffset)*uint64(pageSize)+uint64(blobChunkOffset*BlobChunkSize), pageSize, blobOffsetSlice) //ReadBytes(db, int64(blobOffset)*int64(pageSize)+int64(blobChunkOffset*BlobChunkSize), BlobChunkSize)

	//lang := b[:32] //Language of base, may be affects on index
	numblocks := binary.LittleEndian.Uint32(b[32 : 32+4])
	blocksOfReplacemant := make([]uint32, numblocks)

	for n := 0; n < int(numblocks); n++ {
		blocksOfReplacemant[n] = binary.LittleEndian.Uint32(b[36+n*4 : 36+n*4+4])
	}
	return blocksOfReplacemant
}

func CalcFieldSize(fieldType string, length int) (int, error) {
	var returnLength int
	var err error

	switch fieldType {
	case "B":
		returnLength = length
	case "L":
		returnLength = 1
	case "N":
		returnLength = length
	case "NC":
		returnLength = length * 2
	case "NVC":
		returnLength = length*2 + 2
	case "RV":
		returnLength = 16
	case "NT":
		returnLength = 8
	case "I":
		returnLength = 8
	case "DT":
		returnLength = 7
	default:
		err = errors.New(fmt.Sprintln("Unknown field type ", fieldType))
	}
	return returnLength, err
}

func getTableDescription(s string) (Table, error) {
	var recordLock bool
	var dataFieldOffset int
	var offset int
	var caseSensitive bool
	//var dataOffset, blobOffset, indexOffset int

	result := TableDescriptionPattern.FindStringSubmatch(s)

	if len(result) == 0 {
		return Table{}, errors.New(fmt.Sprint("The format of Table is not valid ", s))
	}

	if result[4] == "1" {
		recordLock = true
	} else {
		recordLock = false
	}

	splitTable := strings.Split(result[5], ",")
	dataOffset, _ := strconv.Atoi(splitTable[0])
	blobOffset, _ := strconv.Atoi(splitTable[1])
	indexOffset, _ := strconv.Atoi(splitTable[2])

	Table := Table{
		Name:        result[1],
		RecordLock:  recordLock,
		DataOffset:  dataOffset,
		BlobOffset:  blobOffset,
		IndexOffset: indexOffset,
		Fields:      make(map[string]Field),
	}

	contain := strings.Contains(result[5], "RV")

	if contain {
		offset = 17
	} else {
		offset = 1
	}

	splitFields := strings.Split(result[2], "\n")

	for _, field_str := range splitFields {
		res := FieldDescriptionPattern.FindStringSubmatch(field_str)
		if len(res) == 0 {
			return Table, errors.New(fmt.Sprint("The format of Field is not valid ", field_str))
		}

		nullExist := false
		if res[3] == "1" {
			nullExist = true
		}

		name := res[1]
		fieldType := res[2]
		lenth, _ := strconv.Atoi(res[4])
		precision, _ := strconv.Atoi(res[5])

		if res[6] == "CS" {
			caseSensitive = true
		} else {
			caseSensitive = false
		}

		dataLength := 0
		if nullExist {
			dataLength = 1
		}
		currentDataL, err := CalcFieldSize(fieldType, lenth)
		if err != nil {
			return Table, err
		}
		dataLength += currentDataL

		if fieldType == "RV" {
			dataFieldOffset = 1
		} else {
			dataFieldOffset = offset
			offset += dataLength
		}

		Field := Field{
			Name:            name,
			FieldType:       fieldType,
			NullExist:       nullExist,
			Lenth:           lenth,
			Precision:       precision,
			CaseSensitive:   caseSensitive,
			DataFieldOffset: dataFieldOffset,
			DataLength:      dataLength,
		}
		Table.Fields[name] = Field
	}
	Table.RowLength = Max(5, offset)

	return Table, nil
}

// Max returns the larger of x or y.
func Max(x, y int) int {
	if x < y {
		return y
	}
	return x
}

type Field struct {
	Name            string
	FieldType       string
	NullExist       bool
	Lenth           int
	Precision       int
	CaseSensitive   bool
	DataFieldOffset int
	DataLength      int
}

func readDataPagesOffsets(BO *BaseOnec) []uint32 {
	pageSize := BO.HeadDB.PageSize
	b := ReadBytes(BO.db, RootObjectOffset*uint64(pageSize), pageSize) //sig := b[:2] //signature of object	//fatLevel := binary.LittleEndian.Uint16(b[2:4])
	lenth := binary.LittleEndian.Uint64(b[16 : 16+8])

	dataPagesCount := int(math.Ceil(float64(lenth) / float64(pageSize)))
	dataPagesOffsets := make([]uint32, dataPagesCount) // Pages of Root Object

	for n := 0; n < dataPagesCount; n++ {
		dataPagesOffsets[n] = binary.LittleEndian.Uint32(b[24+n*4 : 24+n*4+4])
	}

	return dataPagesOffsets
}

func readTableDescriptions(BO *BaseOnec, dataPagesOffsets []uint32, blocksOfReplacemant []uint32) ([]Table, error) {
	var err error
	pageSize := BO.HeadDB.PageSize
	TableDescription := make([]Table, len(blocksOfReplacemant))
	for n, chunkOffset := range blocksOfReplacemant {
		offset := uint64(dataPagesOffsets[uint32(chunkOffset)*BlobChunkSize/pageSize])*uint64(pageSize) + uint64(uint32(chunkOffset)*BlobChunkSize%pageSize)
		if chunkOffset == 0 {
			continue
		}
		TableDescription[n], err = getTableDescription(string(readBlobStream(BO.db, uint64(offset), pageSize, dataPagesOffsets)))
		if err != nil {
			return TableDescription, err
		}
	}
	return TableDescription, nil
}

// Read Root Object
func (BO *BaseOnec) RootObject() error {
	var err error
	dataPagesOffsets := readDataPagesOffsets(BO)

	blocksOfReplacemant := readBlockOfReplacemant(BO, dataPagesOffsets)

	BO.TableDescription, err = readTableDescriptions(BO, dataPagesOffsets, blocksOfReplacemant)
	if err != nil {
		return err
	}
	return nil
}

func DatabaseReader(db *os.File) *BaseOnec {
	BaseOnec := &BaseOnec{
		db: db,
	}
	var err error
	BaseOnec.HeadDB, err = readHeadDB(BaseOnec.db)
	if err != nil {
		log.Fatal("HeadDB read failed", err)
	}
	if BaseOnec.HeadDB.Ver != Ver8380 {
		log.Fatal("Do not support another version", BaseOnec.HeadDB.Ver)
	}
	err = BaseOnec.RootObject()
	if err != nil {
		log.Fatal("RootObject read failed ", err)
	}
	return BaseOnec
}
