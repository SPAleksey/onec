package onec

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"math"
	"os"
)

var Ver8380 = [4]byte{8, 3, 8, 0}

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
	TableDescription []string
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
	rd := bufio.NewReader(db)
	buf, err := rd.Peek(24)
	if err != nil {
		log.Fatal("rd.Read failed", err)
		return headDB, err
	}
	buffer := bytes.NewBuffer(buf)
	err = binary.Read(buffer, binary.LittleEndian, &headDB)
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

func readBlobFirst(db *os.File, pageSize uint32, blobSize uint32, blobOffsetSlice []uint32, blobChunkOffset uint32, fieldType string) []uint32 {
	blobOffset := blobOffsetSlice[0]
	if blobSize == 0 {
		return make([]uint32, 0)
	}
	fmt.Println(int64(blobOffset)*int64(pageSize) + int64(blobChunkOffset*BlobChunkSize))
	b := readBlobStream(db, uint64(blobOffset)*uint64(pageSize)+uint64(blobChunkOffset*BlobChunkSize), pageSize, blobOffsetSlice) //ReadBytes(db, int64(blobOffset)*int64(pageSize)+int64(blobChunkOffset*BlobChunkSize), BlobChunkSize)

	lang := b[:32]
	numblocks := binary.LittleEndian.Uint32(b[32 : 32+4])
	blocksOfReplacemant := make([]uint32, numblocks)
	fmt.Println(string(lang))

	for n := 0; n < int(numblocks); n++ {
		blocksOfReplacemant[n] = binary.LittleEndian.Uint32(b[36+n*4 : 36+n*4+4])

	}

	return blocksOfReplacemant

}

func Table() {

}

// Read Root Object
func (BO *BaseOnec) RootObject() {

	pageSize := BO.HeadDB.PageSize
	b := ReadBytes(BO.db, RootObjectOffset*uint64(pageSize), pageSize)
	//sig := b[:2] //signature of object
	//fatLevel := binary.LittleEndian.Uint16(b[2:4])
	lenth := binary.LittleEndian.Uint64(b[16 : 16+8])

	dataPagesCount := int(math.Ceil(float64(lenth) / float64(pageSize)))
	dataPagesOffsets := make([]uint32, dataPagesCount) // Pages of Root Object

	for n := 0; n < dataPagesCount; n++ {
		dataPagesOffsets[n] = binary.LittleEndian.Uint32(b[24+n*4 : 24+n*4+4])
	}

	blocksOfReplacemant := readBlobFirst(BO.db, pageSize, uint32(lenth), dataPagesOffsets, 1, "I")

	BO.TableDescription = make([]string, len(blocksOfReplacemant))
	for n, chunkOffset := range blocksOfReplacemant {
		offset := uint64(dataPagesOffsets[uint32(chunkOffset)*BlobChunkSize/pageSize])*uint64(pageSize) + uint64(uint32(chunkOffset)*BlobChunkSize%pageSize)
		if chunkOffset == 0 {
			continue
		}
		BO.TableDescription[n] = string(readBlobStream(BO.db, uint64(offset), pageSize, dataPagesOffsets))

	}

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
		log.Fatal("Do not support another version")
	}
	BaseOnec.RootObject()

	return BaseOnec
}

func init() {
	//table_description_pattern_text = '\{"(\S+)".*\n\{"Fields",\n([\s\S]*)\n\},\n\{"Indexes"(?:,|)([\s\S]*)\},' \
	//                                 '\n\{"Recordlock","(\d)+"\},\n\{"Files",(\S+)\}\n\}'
	//table_description_pattern = re.compile(table_description_pattern_text)
	//field_description_pattern = re.compile('\{"(\w+)","(\w+)",(\d+),(\d+),(\d+),"(\w+)"\}(?:,|)')

	//.MustCompile("welcome ([A-z]*) new ([A-z]*) city")

	/*
		{"IBVERSION",0,
		{"Fields",
		{"IBVERSION","N",0,10,0,"CS"},
		{"PLATFORMVERSIONREQ","N",0,10,0,"CS"}
		},
		{"Indexes"},
		{"Recordlock","0"},
		{"Files",20,0,0}
		}	 */

	//Pattern := regexp.MustCompile("{ [A-z]*,\n\{"
	//Fields

}
