package protocgenghe

import (
	"encoding/base32"
	"encoding/binary"
	"strings"
)

type CaptureFnSignature struct {
	PatternByteMapper ByteMapper
	typeMask          uint32
}

func NewCaptureParserSignature(part *URLBarePathPart) *CaptureFnSignature {
	var typeMask uint32
	for idx, goType := range SupportedURLPartTypes {
		if part.CaptureGoTypeVote.Votes[goType] != 0 {
			typeMask |= (1 << idx)
		}
	}
	return &CaptureFnSignature{
		PatternByteMapper: part.PatternByteMapper,
		typeMask:          typeMask,
	}
}

func (sig *CaptureFnSignature) Base32String() string {
	buf := make([]byte, 8+8+4)
	binary.LittleEndian.PutUint64(buf[0:], sig.PatternByteMapper.bits[0])
	binary.LittleEndian.PutUint64(buf[8:], sig.PatternByteMapper.bits[1])
	binary.LittleEndian.PutUint32(buf[16:], sig.typeMask)
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(buf)
}

func (sig *CaptureFnSignature) ReturnGoTypes() (retGoTypes []string) {
	for idx, goType := range SupportedURLPartTypes {
		if (sig.typeMask & (1 << idx)) != 0 {
			retGoTypes = append(retGoTypes, goType)
		}
	}
	return
}

func (sig *CaptureFnSignature) String() string {
	return "[CaptureFnSignature: <" + sig.PatternByteMapper.String() + "> ret={" + strings.Join(sig.ReturnGoTypes(), ",") + "}]"
}

type OrderCaptureFnSig []CaptureFnSignature

func (a OrderCaptureFnSig) Len() int      { return len(a) }
func (a OrderCaptureFnSig) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a OrderCaptureFnSig) Less(i, j int) bool {
	sigI := a[i]
	sigJ := a[j]
	if sigI.PatternByteMapper.bits[0] < sigJ.PatternByteMapper.bits[0] {
		return true
	} else if sigI.PatternByteMapper.bits[0] > sigJ.PatternByteMapper.bits[0] {
		return false
	}
	if sigI.PatternByteMapper.bits[1] < sigJ.PatternByteMapper.bits[1] {
		return true
	} else if sigI.PatternByteMapper.bits[1] > sigJ.PatternByteMapper.bits[1] {
		return false
	}
	return sigI.typeMask < sigJ.typeMask
}
