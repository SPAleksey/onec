package onec

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"math"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"unicode/utf16"
)

var Ver8380 = [4]byte{8, 3, 8, 0}

var (
	TableDescriptionPattern = regexp.MustCompile(`\{"(\S+)".*\n\{"Fields",\n([\s\S]*)\n\},\n\{"Indexes"(?:,|)([\s\S]*)\},\n\{"Recordlock","(\d)+"\},\n\{"Files",(\S+)\}\n\}`)
	FieldDescriptionPattern = regexp.MustCompile(`\{"(\w+)","(\w+)",(\d+),(\d+),(\d+),"(\w+)"\}(?:,|)`)
)

const RootObjectOffset uint64 = 2
const BlobChunkSize uint32 = 256

type BaseOnec struct {
	Db               *os.File
	HeadDB           headDB
	TableDescription map[string]Table
	TablesName       []string
}

type headDB struct { //8s4bIiI
	Cd            [8]byte //  “1CDBMSV8”
	Ver           [4]byte //8 3 8 0
	NumberOfPages int32   //number of pages
	Unknown       uint32  // unknown
	PageSize      uint32  // page size
}

type Table struct {
	Name               string
	RecordLock         bool
	DataOffset         int
	BlobOffset         int
	IndexOffset        int
	RowLength          int
	Fields             map[string]Field
	FieldsName         []string
	NoRecords          bool //0 records of this table in base
	BlockOfReplacemant []uint32
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

type Object struct {
	Table           *Table
	ValueObject     map[string][]byte
	RepresentObject map[string]string
	Number          int //o to ...
	Deleted         bool
}

func ReadBytesOfObject(db *os.File, BlockOfReplacemant []uint32, RowLength int, PageSize int, n int, mu *sync.Mutex) []byte {
	var ToRead, ToEndOfBlock int
	offsetOfNObject := n * RowLength
	LefrToRead := RowLength
	bufTableObject := make([]byte, 0, RowLength)

	for i := 0; LefrToRead > 0; i++ {
		pos := int(BlockOfReplacemant[offsetOfNObject/PageSize+i])*PageSize + (offsetOfNObject % PageSize)
		ToEndOfBlock = PageSize - (offsetOfNObject % PageSize)
		ToRead = Min(LefrToRead, ToEndOfBlock)
		LefrToRead -= ToRead
		buf := ReadBytes(db, uint64(pos), uint32(ToRead), nil)
		bufTableObject = append(bufTableObject, buf...)
	}
	return bufTableObject
}

func (BO *BaseOnec) ReadTableObject(BlockOfReplacemant []uint32, Table Table, n int) Object {

	if Table.NoRecords {
		return Object{}
	}

	Object := Object{
		Table:           &Table,
		ValueObject:     make(map[string][]byte),
		RepresentObject: make(map[string]string),
		Number:          n,
		Deleted:         false,
	}

	bufTableObject := ReadBytesOfObject(BO.Db, BlockOfReplacemant, Table.RowLength, int(BO.HeadDB.PageSize), n, nil)
	deleted := bufTableObject[:1][0] //strconv.Atoi(string(bufTableObject[:1]))

	if deleted == 1 {
		Object.Deleted = true
		return Object
	}
	RepresentObject := make(map[string][]byte)
	for k, v := range Table.Fields {
		RepresentObject[k] = bufTableObject[v.DataFieldOffset:(v.DataFieldOffset + v.DataLength)]
		//fmt.Println(k, " val: ", RepresentObject[k])
		//fmt.Println(k, " val: ", string(RepresentObject[k]))
		//fmt.Println(k, " FieldType: ", v.FieldType)
		value := RepresentObject[k]
		if v.NullExist {
			if value[:1][0] == 0 {
				//value = nil
				//fmt.Println(k, " NullExist FieldType: ", v.FieldType)
			} else {
				value = value[1:]
			}
		}
		Object.ValueObject[k] = value
		Object.RepresentObject[k] = FromFormat1C(value, v.FieldType)
		/*
			if field.null_exists:
			if buffer[:1] == b'\x00':
			# Поле не содержит значения (NULL)
			return None
			else:
			# Обрезаем флаг пустого значения
			buffer = buffer[1:]
		*/

	}
	return Object
}

func FromFormat1C(value []byte, fieldType string) string {
	var returnValue string

	switch fieldType {
	case "NVC":
		lenth := binary.LittleEndian.Uint16(value[:2])
		//fmt.Println("lenth of NVC", lenth)
		var value16 []uint16
		var v16 uint16
		for n := 1; n <= int(lenth); n++ { //len(value)/2-2; n++ {
			v16 = binary.LittleEndian.Uint16(value[n*2 : n*2+3])
			value16 = append(value16, v16)
		}
		enc := utf16.Decode(value16)
		//fmt.Println("value UTF16 of NVC", string(enc))
		returnValue = string(enc)
	default:
		returnValue = string(value)
	}
	return returnValue
}

func (BO *BaseOnec) CheckBlockOfReplacemant(s string) {
	if len(BO.TableDescription[s].BlockOfReplacemant) == 0 && !BO.TableDescription[s].NoRecords {
		BlockOfReplacemant := ReadBlockOfReplacemant(BO, BO.TableDescription[s])
		if len(BlockOfReplacemant) != 0 {
			tempT := BO.TableDescription[s]
			tempT.BlockOfReplacemant = BlockOfReplacemant
			BO.TableDescription[s] = tempT
		}
	}
}

func (BO *BaseOnec) Rows(s string, n int) Object {

	BO.CheckBlockOfReplacemant(s)

	return BO.ReadTableObject(BO.TableDescription[s].BlockOfReplacemant, BO.TableDescription[s], n)
	//return Rows
}

func ReadBytes(db *os.File, position uint64, lenth uint32, mu *sync.Mutex) []byte {
	if mu != nil {
		mu.Lock()
		defer mu.Unlock()
	}
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
	rd := bufio.NewReader(Db)
	buf, err := rd.Peek(24)
	if err != nil {
		log.Fatal("rd.Read failed", err)
		return headDB, err
	}
	fmt.Println(buf) // - takes on different values on benchmark
	*/
	buf := ReadBytes(db, 0, 24, nil)
	buffer := bytes.NewBuffer(buf)
	err := binary.Read(buffer, binary.LittleEndian, &headDB)
	if err != nil {
		log.Fatal("binary.Read failed", err)
		return headDB, err
	}

	return headDB, nil
}

func readBlobStream(db *os.File, offset uint64, pageSize uint32, dataPagesOffsets []uint32, mu *sync.Mutex) []byte {
	nextBlock := uint32(1)
	data := make([]byte, 0, 250)
	currentOffset := offset
	for nextBlock != 0 {
		b := ReadBytes(db, currentOffset, BlobChunkSize, mu)
		nextBlock = binary.LittleEndian.Uint32(b[:4])
		nextPage := uint32(nextBlock) * BlobChunkSize / pageSize
		currentOffset = uint64(dataPagesOffsets[nextPage])*uint64(pageSize) + uint64(uint32(nextBlock)*BlobChunkSize%pageSize)
		size := binary.LittleEndian.Uint16(b[4:6])
		data = append(data, b[6:6+size]...)
	}

	return data
}

func readBlockOfReplacemantRoot(BO *BaseOnec, blobOffsetSlice []uint32) []uint32 {
	var blobChunkOffset uint32 = 1
	pageSize := BO.HeadDB.PageSize

	blobOffset := blobOffsetSlice[0]
	//if blobSize == 0 {return make([]uint32, 0)}
	//fmt.Println(int64(blobOffset)*int64(pageSize) + int64(blobChunkOffset*BlobChunkSize))
	b := readBlobStream(BO.Db, uint64(blobOffset)*uint64(pageSize)+uint64(blobChunkOffset*BlobChunkSize), pageSize, blobOffsetSlice, nil) //ReadBytes(Db, int64(blobOffset)*int64(pageSize)+int64(blobChunkOffset*BlobChunkSize), BlobChunkSize)

	//lang := b[:32] //Language of base, may be affects on index
	numblocks := binary.LittleEndian.Uint32(b[32 : 32+4])
	blocksOfReplacemant := make([]uint32, numblocks)

	for n := 0; n < int(numblocks); n++ {
		blocksOfReplacemant[n] = binary.LittleEndian.Uint32(b[36+n*4 : 36+n*4+4])
	}
	return blocksOfReplacemant
}

func ReadBlockOfReplacemant(BO *BaseOnec, table Table) []uint32 {

	pageSize := BO.HeadDB.PageSize
	offset := uint64(pageSize) * uint64(table.DataOffset)

	/*
	   	struct {
	   		unsigned int object_type; //0xFD1C или 0x01FD1C
	   		unsigned int version1;
	   		unsigned int version2;
	   		unsigned int version3;
	   		unsigned long int length; //64-разрядное целое!
	   		unsigned int pages[];
	   	}
	      first 5 filed = 24 bytes
	*/

	buf := ReadBytes(BO.Db, offset, pageSize, nil)
	//a0 := [2]byte(buf)
	//fatLevel := [1]byte(buf[2:3])

	sig := hex.EncodeToString(buf[:2])
	if sig == "1cfd" {
	}
	fatLevel := buf[2:3][0] //fatLevel, _ :=strconv.Atoi(string(buf[2:3]))
	lenth := binary.LittleEndian.Uint64(buf[16:25])
	numberOfBlocks := int(math.Ceil(float64(lenth) / float64(pageSize)))
	blocksOfReplacemant := make([]uint32, numberOfBlocks)
	if fatLevel == 0 {
		for n := 0; n < numberOfBlocks; n++ {
			blocksOfReplacemant[n] = binary.LittleEndian.Uint32(buf[24+n*4 : 25+(n+1)*4])
		}
	} else if fatLevel == 1 {
		index_pages_offsets := make([]uint32, 2)
		for n := 0; ; n++ {
			pages_offset := binary.LittleEndian.Uint32(buf[24+n*4 : 25+(n+1)*4])
			if pages_offset == 0 {
				break
			}
			index_pages_offsets[n] = pages_offset
		}
		for n := 0; n < len(index_pages_offsets); n++ {
			buf = ReadBytes(BO.Db, uint64(index_pages_offsets[n])*uint64(pageSize), pageSize, nil)
			for i := 0; i < len(buf)/4; i++ {
				block := binary.LittleEndian.Uint32(buf[n*4 : 1+(n+1)*4])
				if block == 0 {
					break
				}
				blocksOfReplacemant[n*int(pageSize/4)+i] = block
			}
		}
	} else {
		panic("unknown fatlevel")
	}

	_ = sig
	//fmt.Println("2 bytes ", (buf[2:3]))
	//fmt.Println("2 bytes ", hex.EncodeToString((buf[:2])))
	if len(blocksOfReplacemant) == 0 { //0 records in table
		TempT := BO.TableDescription[table.Name]
		TempT.NoRecords = true
		BO.TableDescription[table.Name] = TempT
		//table.NoRecords = true
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
		returnLength = length/2 + 1
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
		Name:               result[1],
		RecordLock:         recordLock,
		DataOffset:         dataOffset,
		BlobOffset:         blobOffset,
		IndexOffset:        indexOffset,
		Fields:             make(map[string]Field),
		FieldsName:         []string{},
		BlockOfReplacemant: make([]uint32, 0, 0),
	}

	contain := strings.Contains(result[5], "RV")

	if contain {
		offset = 17
	} else {
		offset = 1
	}

	splitFields := strings.Split(result[2], "\n")
	TableFieldsName := make([]string, 0, len(splitFields))

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
		TableFieldsName = append(TableFieldsName, name)
	}
	Table.RowLength = Max(5, offset)
	sort.Strings(TableFieldsName)
	Table.FieldsName = TableFieldsName

	return Table, nil
}

// Max returns the larger of x or y.
func Max(x, y int) int {
	if x < y {
		return y
	}
	return x
}

func Min(x, y int) int {
	if x > y {
		return y
	}
	return x
}

func readDataPagesOffsets(BO *BaseOnec) []uint32 {
	pageSize := BO.HeadDB.PageSize
	b := ReadBytes(BO.Db, RootObjectOffset*uint64(pageSize), pageSize, nil) //sig := b[:2] //signature of object	//fatLevel := binary.LittleEndian.Uint16(b[2:4])
	lenth := binary.LittleEndian.Uint64(b[16 : 16+8])

	dataPagesCount := int(math.Ceil(float64(lenth) / float64(pageSize)))
	dataPagesOffsets := make([]uint32, dataPagesCount) // Pages of Root Object

	for n := 0; n < dataPagesCount; n++ {
		dataPagesOffsets[n] = binary.LittleEndian.Uint32(b[24+n*4 : 24+n*4+4])
	}

	return dataPagesOffsets
}

func readTablesDescriptions(BO *BaseOnec, dataPagesOffsets []uint32, blocksOfReplacemant []uint32, mu *sync.Mutex) (map[string]Table, []string, error) {
	//var err error
	tablesChan := make(chan Table)
	defer close(tablesChan)

	wg := new(sync.WaitGroup)
	pageSize := BO.HeadDB.PageSize
	TablesDescription := make(map[string]Table)
	TablesName := make([]string, len(blocksOfReplacemant))

	for _, chunkOffset := range blocksOfReplacemant {
		if chunkOffset == 0 {
			continue
		}

		//run goroutines for each table
		wg.Add(1)
		go func(db *os.File, chunkOffset uint32, pageSize uint32, dataPagesOffsets []uint32, tablesChan chan<- Table, wg *sync.WaitGroup) {
			defer wg.Done()
			offset := uint64(dataPagesOffsets[uint32(chunkOffset)*BlobChunkSize/pageSize])*uint64(pageSize) + uint64(uint32(chunkOffset)*BlobChunkSize%pageSize)
			tablesDescription, err := getTableDescription(string(readBlobStream(db, uint64(offset), pageSize, dataPagesOffsets, mu)))
			if err != nil {
				tablesDescription = Table{Name: fmt.Sprint("offset ", offset, " page ", pageSize)}
				fmt.Println(err)
				//return err
			}
			tablesChan <- tablesDescription
		}(BO.Db, chunkOffset, pageSize, dataPagesOffsets, tablesChan, wg)
	}

	//read from chan tableDescription
	go func(TablesDescription map[string]Table, tablesChan <-chan Table) {
		i := 0
		for tableDescription := range tablesChan {
			TablesDescription[tableDescription.Name] = tableDescription
			TablesName[i] = tableDescription.Name
			i++
		}
	}(TablesDescription, tablesChan)

	wg.Wait()
	sort.Strings(TablesName)
	return TablesDescription, TablesName, nil
}

// Read Root Object
func (BO *BaseOnec) RootObject(mu *sync.Mutex) error {
	var err error
	dataPagesOffsets := readDataPagesOffsets(BO)

	blocksOfReplacemant := readBlockOfReplacemantRoot(BO, dataPagesOffsets)

	BO.TableDescription, BO.TablesName, err = readTablesDescriptions(BO, dataPagesOffsets, blocksOfReplacemant, mu)

	if err != nil {
		return err
	}
	return nil
}

func DatabaseReader(db *os.File) (*BaseOnec, error) {
	BaseOnec := &BaseOnec{
		Db: db,
	}
	var err error
	mu := new(sync.Mutex)
	BaseOnec.HeadDB, err = readHeadDB(BaseOnec.Db)
	if err != nil {
		return nil, err
		//log.Fatal("HeadDB read failed", err)
	}
	if BaseOnec.HeadDB.Ver != Ver8380 {
		//return nil, errors.New("Do not support another version, this ver: "+ string(BaseOnec.HeadDB.Ver))
		log.Fatal("Do not support another version", BaseOnec.HeadDB.Ver)
	}
	err = BaseOnec.RootObject(mu)
	if err != nil {
		return nil, err
		//log.Fatal("RootObject read failed ", err)
	}
	return BaseOnec, nil
}

func OpenBaseOnec(path string) (*BaseOnec, error) {
	db, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	//	defer Db.Close()

	BaseOnec, err := DatabaseReader(db)
	if err != nil {
		return nil, err
	}

	return BaseOnec, nil
}
