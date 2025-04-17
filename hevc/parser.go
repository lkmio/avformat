package hevc

//
//func (self HEVCDecoderConfRecord) Marshal(b []byte, si HEVCSPSInfo) (n int) {
//	b[0] = 1
//	b[1] = self.AVCProfileIndication
//	b[2] = self.ProfileCompatibility
//	b[3] = self.AVCLevelIndication
//	b[21] = 3
//	b[22] = 3
//	n += 23
//	b[n] = (self.VPS[0][0] >> 1) & 0x3f
//	n++
//	b[n] = byte(len(self.VPS) >> 8)
//	n++
//	b[n] = byte(len(self.VPS))
//	n++
//	for _, vps := range self.VPS {
//		binary.BigEndian.PutUint16(b[n:], uint16(len(vps)))
//		n += 2
//		copy(b[n:], vps)
//		n += len(vps)
//	}
//	b[n] = (self.SPS[0][0] >> 1) & 0x3f
//	n++
//	b[n] = byte(len(self.SPS) >> 8)
//	n++
//	b[n] = byte(len(self.SPS))
//	n++
//	for _, sps := range self.SPS {
//		binary.BigEndian.PutUint16(b[n:], uint16(len(sps)))
//		n += 2
//		copy(b[n:], sps)
//		n += len(sps)
//	}
//	b[n] = (self.PPS[0][0] >> 1) & 0x3f
//	n++
//	b[n] = byte(len(self.PPS) >> 8)
//	n++
//	b[n] = byte(len(self.PPS))
//	n++
//	for _, pps := range self.PPS {
//		binary.BigEndian.PutUint16(b[n:], uint16(len(pps)))
//		n += 2
//		copy(b[n:], pps)
//		n += len(pps)
//	}
//	return
//}
