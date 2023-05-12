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
	HashesUsersPattern      = regexp.MustCompile(`\d+,\d+,"(\S+)","(\S+)",\d+,\d+`)
)

const RootObjectOffset uint64 = 2
const BlobChunkSize uint32 = 256
const BlobChunkOffset uint32 = 1

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
	Name        string
	RecordLock  bool
	DataOffset  int
	BlobOffset  int
	IndexOffset int
	RowLength   int
	Fields      map[string]Field
	FieldsName  []string
	//NoRecords              bool //0 records of this table in base
	BlockOfReplacemant     []uint32
	BlockOfReplacemantBlob []uint32
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
	NotExist        bool
	//BlobData        []byte
}

func ReadBytesOfObject(db *os.File, BlockOfReplacemant []uint32, RowLength int, PageSize int, n int, mu *sync.Mutex) []byte {
	var ToRead, ToEndOfBlock, pos int
	offsetOfNObject := n * RowLength
	LefrToRead := RowLength
	bufTableObject := make([]byte, 0, RowLength)

	if offsetOfNObject/PageSize > len(BlockOfReplacemant) { //out of file
		fmt.Println("out1 ", n)
		return []byte{}
	}

	for i := 0; LefrToRead > 0; i++ {
		//error??
		if (offsetOfNObject/PageSize + i) >= len(BlockOfReplacemant) { //out of file
			fmt.Println("out2 ", n)
			return []byte{}
		}
		if i == 0 {
			pos = int(BlockOfReplacemant[offsetOfNObject/PageSize])*PageSize + (offsetOfNObject % PageSize)
		} else {
			pos = int(BlockOfReplacemant[offsetOfNObject/PageSize+i]) * PageSize
		}

		if i == 0 {
			ToEndOfBlock = PageSize - (offsetOfNObject % PageSize)
		} else {
			ToEndOfBlock = PageSize
		}
		ToRead = Min(LefrToRead, ToEndOfBlock)
		LefrToRead -= ToRead
		buf := ReadBytes(db, uint64(pos), uint32(ToRead), nil)
		bufTableObject = append(bufTableObject, buf...)
	}
	return bufTableObject
}

func (BO *BaseOnec) ReadTableObject(BlockOfReplacemant []uint32, Table Table, n int, blobValue bool) Object {

	if len(Table.BlockOfReplacemant) == 0 { //no record
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

	if len(bufTableObject) == 0 || allZero(bufTableObject) {
		Object.NotExist = true
		return Object
	}

	deleted := bufTableObject[:1][0] //strconv.Atoi(string(bufTableObject[:1]))
	if deleted == 1 {
		Object.Deleted = true
		return Object
	}

	//if Table.BlobOffset != 0 {
	//	Object.BlobData = readBlobStream(BO.Db, uint64(Table.BlockOfReplacemantBlob[0])*uint64(BO.HeadDB.PageSize)+uint64(BlobChunkOffset*BlobChunkSize), BO.HeadDB.PageSize, Table.BlockOfReplacemantBlob, nil)
	//}

	for k, v := range Table.Fields {
		value := bufTableObject[v.DataFieldOffset:(v.DataFieldOffset + v.DataLength)] //RepresentObject[k]
		Object.ValueObject[k] = value
		Object.RepresentObject[k] = FromFormat1C(value, v, &Object, BO, blobValue)
	}
	return Object
}

func allZero(s []byte) bool {
	for _, v := range s {
		if v != 0 {
			return false
		}
	}
	return true
}

func FromFormat1C(value []byte, field Field, object *Object, BO *BaseOnec, blobValue bool) string {
	var returnValue string

	if field.NullExist {
		if int(value[0]) == 0 {
			return "" //Поле не содержит значения (NULL)
		}
		value = value[1:] //Обрезаем флаг пустого значения
	}

	switch field.FieldType {
	case "NVC": //«NVC» - строка переменной длины. Длина поля равна FieldLength * 2 + 2 байт. Первые 2 байта содержат длину строки (максимум FieldLength). Оставшиеся байты представляет собой строку в формате Unicode (каждый символ занимает 2 байта).
		lenth := binary.LittleEndian.Uint16(value[:2])

		if lenth > uint16(field.Lenth) {
			lenth = uint16(field.Lenth)
		}

		var value16 []uint16
		for n := 1; n <= int(lenth); n++ { //len(value)/2-2; n++ {
			value16 = append(value16, binary.LittleEndian.Uint16(value[n*2:n*2+2]))
		}
		returnValue = string(utf16.Decode(value16))
	case "NC":
		var value16 []uint16
		for n := 0; n < len(value); n++ { //len(value)/2-2; n++ {
			value16 = append(value16, binary.LittleEndian.Uint16(value[n*2:n*2+2]))
		}
		returnValue = string(utf16.Decode(value16))
	case "DT": //«DT» - дата-время. Длина поля 7 байт. Содержит данные в двоично-десятичном виде. Первые 2 байта содержат четыре цифры года, третий байт – две цифры месяца, четвертый байт – день, пятый – часы, шестой – минуты и седьмой – секунды, все также по 2 цифры.
		d := make([]string, 0, 7)
		var t string
		for _, b := range value {
			t = strconv.FormatInt(int64(b), 16)
			if len(t) == 1 {
				t = "0" + t
			}
			d = append(d, t)
		}
		returnValue = d[0] + d[1] + "." + d[2] + "." + d[3] + " " + d[4] + ":" + d[5] + ":" + d[6]
	case "N": //«N» - число. Длина поля в байтах равна Цел((FieldLength + 2) / 2). Числа хранятся в двоично-десятичном виде. Первый полубайт означает знак числа. 0 – число отрицательное, 1 – положительное. Каждый следующий полубайт соответствует одной десятичной цифре. Всего цифр FieldLength. Десятичная точка находится в FieldPrecision цифрах справа. Например, FieldLength = 5, FieldPrecision = 3. Байты 0x18, 0x47, 0x23 означают число 84.723, а байты 0x00, 0x00, 0x91 представляют число -0.091.
		Ib, _ := strconv.Atoi(strconv.FormatInt(int64(value[0]), 16))
		sign := ((Ib/10)*2 - 1)
		returnValueF := float64(sign * Ib % 10)
		//for i := 1; i < field.Lenth/2+1; i++ {
		for i := 1; i < len(value); i++ {
			Ib, _ = strconv.Atoi(strconv.FormatInt(int64(value[i]), 16))
			returnValueF = returnValueF*100 + float64(Ib) // float64(Ib/10)*10 + float64(Ib%10)
		}
		returnValueF = returnValueF / math.Pow10(field.Precision+1)
		returnValue = strconv.FormatFloat(returnValueF, 'f', -1, 64) //fmt.Sprintf("%f", returnValueF)
	case "L":
		if value[0] == 0 {
			return "false"
		}
		return "true"
	case "I", "NT":
		ChunkOffset := binary.LittleEndian.Uint32(value[:4])
		LenthBlob := binary.LittleEndian.Uint32(value[4:])

		if blobValue {
			rv := ReadBlobStream(BO.Db, uint64(object.Table.BlockOfReplacemantBlob[uint32(ChunkOffset)*BlobChunkSize/BO.HeadDB.PageSize])*uint64(BO.HeadDB.PageSize)+uint64(ChunkOffset*BlobChunkSize%BO.HeadDB.PageSize), BO.HeadDB.PageSize, object.Table.BlockOfReplacemantBlob, nil)
			returnValue = string(rv[:LenthBlob])
			if len(rv) < int(LenthBlob) {
				LenthBlob = uint32(len(rv))
			}
			returnValue = string(rv[:LenthBlob])
		} else {
			returnValue = strings.Join([]string{"/blob/", strconv.Itoa(object.Table.BlobOffset), "/", strconv.Itoa(int(ChunkOffset)), "/", strconv.Itoa(int(LenthBlob))}, "")
		}

		/*
			//rv := rvb[:LenthBlob]
			var value16 []uint16
			for n := 0; n < (len(rv)/2 - 2); n++ { //len(value)/2-2; n++ {
				value16 = append(value16, binary.LittleEndian.Uint16(rv[n*2:n*2+2]))
			}
			returnValue = string(utf16.Decode(value16))
		*/

	default:
		returnValue = ByteSliceToHexString(value)
	}
	return returnValue
}

func ExtractHashes(b []byte) []string {
	result := HashesUsersPattern.FindStringSubmatch(string(b))

	return result
}

func ByteSliceToHexString(originalBytes []byte) string {
	ru := []rune("")
	for _, b := range originalBytes {
		st := strconv.FormatInt(int64(b), 16)
		if len(st) == 1 {
			st = "0" + st
		}
		ru = append(ru, []rune(" 0x"+st)...)
	}

	return string(ru)
}

func ByteSliceToHexString1(originalBytes []byte) string {
	result := make([]byte, 4*len(originalBytes))

	buff := bytes.NewBuffer(result)

	for _, b := range originalBytes {
		//fmt.Println(b)
		//output := strconv.FormatInt(int64(b), 16)
		//fmt.Println("The hexadecimal conversion of", b, "is", output)
		fmt.Fprintf(buff, "0x%02x ", b)
	}

	//fmt.Println(buff.String())
	return buff.String()
}

func (BO *BaseOnec) CheckBlockOfReplacemant(s string) {

	if BO.TableDescription[s].BlockOfReplacemant == nil {
		BlockOfReplacemant := ReadBlockOfReplacemant(BO, BO.TableDescription[s].DataOffset)
		tempT := BO.TableDescription[s]
		tempT.BlockOfReplacemant = BlockOfReplacemant
		BO.TableDescription[s] = tempT
	}

	if len(BO.TableDescription[s].BlockOfReplacemantBlob) == 0 && len(BO.TableDescription[s].BlockOfReplacemant) > 0 && BO.TableDescription[s].BlobOffset != 0 {
		BlockOfReplacemantBlob := ReadBlockOfReplacemant(BO, BO.TableDescription[s].BlobOffset)
		if len(BlockOfReplacemantBlob) != 0 {
			tempT := BO.TableDescription[s]
			tempT.BlockOfReplacemantBlob = BlockOfReplacemantBlob
			BO.TableDescription[s] = tempT
		}
	}

}

func (BO *BaseOnec) Rows(s string, n int, blobValue bool) Object {
	BO.CheckBlockOfReplacemant(s)
	return BO.ReadTableObject(BO.TableDescription[s].BlockOfReplacemant, BO.TableDescription[s], n, blobValue)
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
	buf := ReadBytes(db, 0, 24, nil)
	buffer := bytes.NewBuffer(buf)
	err := binary.Read(buffer, binary.LittleEndian, &headDB)
	if err != nil {
		log.Fatal("binary.Read failed", err)
		return headDB, err
	}

	return headDB, nil
}

func ReadBlobStream(db *os.File, offset uint64, pageSize uint32, dataPagesOffsets []uint32, mu *sync.Mutex) []byte {
	nextBlock := uint32(1)
	data := make([]byte, 0, 250)
	currentOffset := offset
	for nextBlock != 0 {
		b := ReadBytes(db, currentOffset, BlobChunkSize, mu)
		nextBlock = binary.LittleEndian.Uint32(b[:4])
		nextPage := uint32(nextBlock) * BlobChunkSize / pageSize

		if int(nextPage) > len(dataPagesOffsets) { //for del
			fmt.Println("nextPage>888 ")
			return []byte{}
		}
		currentOffset = uint64(dataPagesOffsets[nextPage])*uint64(pageSize) + uint64(uint32(nextBlock)*BlobChunkSize%pageSize)
		size := binary.LittleEndian.Uint16(b[4:6])
		if size > 250 { //for del
			size = uint16(len(b)) - 6
			fmt.Println("size>250 ")
			return []byte{}
		}
		data = append(data, b[6:6+size]...)
	}

	return data
}

func readBlockOfReplacemantRoot(BO *BaseOnec, blobOffsetSlice []uint32) []uint32 {
	pageSize := BO.HeadDB.PageSize

	blobOffset := blobOffsetSlice[0]
	b := ReadBlobStream(BO.Db, uint64(blobOffset)*uint64(pageSize)+uint64(BlobChunkOffset*BlobChunkSize), pageSize, blobOffsetSlice, nil) //ReadBytes(Db, int64(blobOffset)*int64(pageSize)+int64(blobChunkOffset*BlobChunkSize), BlobChunkSize)

	//lang := b[:32] //Language of base, may be affects on index
	numblocks := binary.LittleEndian.Uint32(b[32 : 32+4])
	blocksOfReplacemant := make([]uint32, numblocks)

	for n := 0; n < int(numblocks); n++ {
		blocksOfReplacemant[n] = binary.LittleEndian.Uint32(b[36+n*4 : 36+n*4+4])
	}
	return blocksOfReplacemant
}

func ReadBlockOfReplacemant(BO *BaseOnec, dataOffset int) []uint32 {

	pageSize := BO.HeadDB.PageSize
	offset := uint64(pageSize) * uint64(dataOffset)

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
	if sig != "1cfd" {
		fmt.Println("sig??? ", sig)
	}
	fatLevel := buf[2:3][0] //fatLevel, _ :=strconv.Atoi(string(buf[2:3]))
	lenth := binary.LittleEndian.Uint64(buf[16:25])
	numberOfBlocks := int(math.Ceil(float64(lenth) / float64(pageSize)))
	blocksOfReplacemant := make([]uint32, 0, numberOfBlocks)
	if fatLevel == 0 {
		for n := 0; n < numberOfBlocks; n++ {
			blocksOfReplacemant = append(blocksOfReplacemant, binary.LittleEndian.Uint32(buf[24+n*4:24+(n+1)*4]))
		}
	} else if fatLevel == 1 {
		index_pages_offsets := make([]uint32, 0, 2) //need correct - mistake
		for n := 0; ; n++ {
			pages_offset := binary.LittleEndian.Uint32(buf[24+n*4 : 25+(n+1)*4])
			if pages_offset == 0 {
				break
			}
			index_pages_offsets = append(index_pages_offsets, pages_offset)
		}
		for n := 0; n < len(index_pages_offsets); n++ {
			buf = ReadBytes(BO.Db, uint64(index_pages_offsets[n])*uint64(pageSize), pageSize, nil)
			for i := 0; i < len(buf)/4; i++ {
				block := binary.LittleEndian.Uint32(buf[i*4 : i*4+4])
				if block == 0 {
					break
				}
				blocksOfReplacemant = append(blocksOfReplacemant, block)
			}
		}
	} else {
		panic("unknown fatlevel")
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
		err = errors.New(strings.Join([]string{"Unknown field type ", fieldType}, " "))
	}
	return returnLength, err
}

func getTableDescription(s string) (Table, error) {
	var recordLock bool
	var dataFieldOffset int
	var offset int
	var caseSensitive bool

	result := TableDescriptionPattern.FindStringSubmatch(s)

	if len(result) == 0 {
		return Table{}, errors.New(strings.Join([]string{"The format of Table is not valid ", s}, " "))
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
		BlockOfReplacemant: nil, //make([]uint32, 0, 0),
	}

	//If exist field type "RV" than it fist
	contain := strings.Contains(result[2], "RV")
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
			return Table, errors.New(strings.Join([]string{"The format of Field is not valid ", field_str}, " "))
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

	wg := new(sync.WaitGroup)
	wg1 := new(sync.WaitGroup)
	pageSize := BO.HeadDB.PageSize
	TablesDescription := make(map[string]Table)
	TablesName := make([]string, 0, len(blocksOfReplacemant))

	for _, chunkOffset := range blocksOfReplacemant {
		if chunkOffset == 0 {
			continue
		}

		//run goroutines for each table
		wg.Add(1)
		go func(db *os.File, chunkOffset uint32, pageSize uint32, dataPagesOffsets []uint32, tablesChan chan<- Table, wg *sync.WaitGroup) {
			defer wg.Done()
			offset := uint64(dataPagesOffsets[uint32(chunkOffset)*BlobChunkSize/pageSize])*uint64(pageSize) + uint64(uint32(chunkOffset)*BlobChunkSize%pageSize)
			tablesDescription, err := getTableDescription(string(ReadBlobStream(db, uint64(offset), pageSize, dataPagesOffsets, mu)))
			if err != nil {
				tablesDescription = Table{Name: strings.Join([]string{"offset", strconv.FormatUint(offset, 10), "page", strconv.FormatUint(uint64(pageSize), 10)}, " ")}
				fmt.Println(err)
				//return err
			}
			tablesChan <- tablesDescription
		}(BO.Db, chunkOffset, pageSize, dataPagesOffsets, tablesChan, wg)
	}

	//read from chan tableDescription
	wg1.Add(1)
	go func(TablesDescription map[string]Table, tablesChan <-chan Table) {
		defer wg1.Done()
		for tableDescription := range tablesChan {
			TablesDescription[tableDescription.Name] = tableDescription
			TablesName = append(TablesName, tableDescription.Name)
		}
	}(TablesDescription, tablesChan)

	wg.Wait()
	close(tablesChan)
	wg1.Wait()

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

func OpenBaseOnec(db *os.File) (*BaseOnec, error) {
	//db, err := os.Open(path)
	//if err != nil {
	//	return nil, err
	//}
	//	defer Db.Close()

	BaseOnec, err := DatabaseReader(db)
	if err != nil {
		return nil, err
	}

	return BaseOnec, nil
}
