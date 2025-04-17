package avc

import (
	"bytes"
	"fmt"
	"github.com/lkmio/avformat/bufio"
	"math"
)

type SPS struct {
	Id                uint
	ProfileIdc        uint
	LevelIdc          uint
	ConstraintSetFlag uint

	MbWidth  uint
	MbHeight uint

	CropLeft   uint
	CropRight  uint
	CropTop    uint
	CropBottom uint

	Width  int
	Height int
	FPS    int
}

func ParseSPS(data []byte) (s SPS, err error) {
	data = RemoveStartCode(data)
	r := &bufio.GolombBitReader{R: bytes.NewReader(data)}

	if _, err = r.ReadBits(8); err != nil {
		return
	}

	if s.ProfileIdc, err = r.ReadBits(8); err != nil {
		return
	}

	// constraint_set0_flag-constraint_set6_flag,reserved_zero_2bits
	if s.ConstraintSetFlag, err = r.ReadBits(8); err != nil {
		return
	}
	s.ConstraintSetFlag = s.ConstraintSetFlag >> 2

	// level_idc
	if s.LevelIdc, err = r.ReadBits(8); err != nil {
		return
	}

	// seq_parameter_set_id
	if s.Id, err = r.ReadExponentialGolombCode(); err != nil {
		return
	}

	if s.ProfileIdc == 100 || s.ProfileIdc == 110 ||
		s.ProfileIdc == 122 || s.ProfileIdc == 244 ||
		s.ProfileIdc == 44 || s.ProfileIdc == 83 ||
		s.ProfileIdc == 86 || s.ProfileIdc == 118 {

		var chroma_format_idc uint
		if chroma_format_idc, err = r.ReadExponentialGolombCode(); err != nil {
			return
		}

		if chroma_format_idc == 3 {
			// residual_colour_transform_flag
			if _, err = r.ReadBit(); err != nil {
				return
			}
		}

		// bit_depth_luma_minus8
		if _, err = r.ReadExponentialGolombCode(); err != nil {
			return
		}
		// bit_depth_chroma_minus8
		if _, err = r.ReadExponentialGolombCode(); err != nil {
			return
		}
		// qpprime_y_zero_transform_bypass_flag
		if _, err = r.ReadBit(); err != nil {
			return
		}

		var seq_scaling_matrix_present_flag uint
		if seq_scaling_matrix_present_flag, err = r.ReadBit(); err != nil {
			return
		}

		if seq_scaling_matrix_present_flag != 0 {
			for i := 0; i < 8; i++ {
				var seq_scaling_list_present_flag uint
				if seq_scaling_list_present_flag, err = r.ReadBit(); err != nil {
					return
				}
				if seq_scaling_list_present_flag != 0 {
					var sizeOfScalingList uint
					if i < 6 {
						sizeOfScalingList = 16
					} else {
						sizeOfScalingList = 64
					}
					lastScale := uint(8)
					nextScale := uint(8)
					for j := uint(0); j < sizeOfScalingList; j++ {
						if nextScale != 0 {
							var delta_scale uint
							if delta_scale, err = r.ReadSE(); err != nil {
								return
							}
							nextScale = (lastScale + delta_scale + 256) % 256
						}
						if nextScale != 0 {
							lastScale = nextScale
						}
					}
				}
			}
		}
	}

	// log2_max_frame_num_minus4
	if _, err = r.ReadExponentialGolombCode(); err != nil {
		return
	}

	var pic_order_cnt_type uint
	if pic_order_cnt_type, err = r.ReadExponentialGolombCode(); err != nil {
		return
	}
	if pic_order_cnt_type == 0 {
		// log2_max_pic_order_cnt_lsb_minus4
		if _, err = r.ReadExponentialGolombCode(); err != nil {
			return
		}
	} else if pic_order_cnt_type == 1 {
		// delta_pic_order_always_zero_flag
		if _, err = r.ReadBit(); err != nil {
			return
		}
		// offset_for_non_ref_pic
		if _, err = r.ReadSE(); err != nil {
			return
		}
		// offset_for_top_to_bottom_field
		if _, err = r.ReadSE(); err != nil {
			return
		}
		var num_ref_frames_in_pic_order_cnt_cycle uint
		if num_ref_frames_in_pic_order_cnt_cycle, err = r.ReadExponentialGolombCode(); err != nil {
			return
		}
		for i := uint(0); i < num_ref_frames_in_pic_order_cnt_cycle; i++ {
			if _, err = r.ReadSE(); err != nil {
				return
			}
		}
	}

	// max_num_ref_frames
	if _, err = r.ReadExponentialGolombCode(); err != nil {
		return
	}

	// gaps_in_frame_num_value_allowed_flag
	if _, err = r.ReadBit(); err != nil {
		return
	}

	if s.MbWidth, err = r.ReadExponentialGolombCode(); err != nil {
		return
	}
	s.MbWidth++

	if s.MbHeight, err = r.ReadExponentialGolombCode(); err != nil {
		return
	}
	s.MbHeight++

	var frame_mbs_only_flag uint
	if frame_mbs_only_flag, err = r.ReadBit(); err != nil {
		return
	}
	if frame_mbs_only_flag == 0 {
		// mb_adaptive_frame_field_flag
		if _, err = r.ReadBit(); err != nil {
			return
		}
	}

	// direct_8x8_inference_flag
	if _, err = r.ReadBit(); err != nil {
		return
	}

	var frame_cropping_flag uint
	if frame_cropping_flag, err = r.ReadBit(); err != nil {
		return
	}
	if frame_cropping_flag != 0 {
		if s.CropLeft, err = r.ReadExponentialGolombCode(); err != nil {
			return
		}
		if s.CropRight, err = r.ReadExponentialGolombCode(); err != nil {
			return
		}
		if s.CropTop, err = r.ReadExponentialGolombCode(); err != nil {
			return
		}
		if s.CropBottom, err = r.ReadExponentialGolombCode(); err != nil {
			return
		}
	}

	s.Width = int((s.MbWidth * 16) - s.CropLeft*2 - s.CropRight*2)
	s.Height = int(((2 - frame_mbs_only_flag) * s.MbHeight * 16) - s.CropTop*2 - s.CropBottom*2)

	vui_parameter_present_flag, err := r.ReadBit()
	if err != nil {
		return
	}

	if vui_parameter_present_flag != 0 {
		aspect_ratio_info_present_flag, err := r.ReadBit()
		if err != nil {
			return s, err
		}

		if aspect_ratio_info_present_flag != 0 {
			aspect_ratio_idc, err := r.ReadBits(8)
			if err != nil {
				return s, err
			}

			if aspect_ratio_idc == 255 {
				sar_width, err := r.ReadBits(16)
				if err != nil {
					return s, err
				}
				sar_height, err := r.ReadBits(16)
				if err != nil {
					return s, err
				}

				_, _ = sar_width, sar_height
			}
		}

		overscan_info_present_flag, err := r.ReadBit()
		if err != nil {
			return s, err
		}

		if overscan_info_present_flag != 0 {
			overscan_appropriate_flagu, err := r.ReadBit()
			if err != nil {
				return s, err
			}

			_ = overscan_appropriate_flagu
		}
		video_signal_type_present_flag, err := r.ReadBit()
		if video_signal_type_present_flag != 0 {
			video_format, err := r.ReadBits(3)
			if err != nil {
				return s, err
			}
			_ = video_format
			video_full_range_flag, err := r.ReadBit()
			if err != nil {
				return s, err
			}
			_ = video_full_range_flag
			colour_description_present_flag, err := r.ReadBit()
			if err != nil {
				return s, err
			}
			if colour_description_present_flag != 0 {
				colour_primaries, err := r.ReadBits(8)
				if err != nil {
					return s, err
				}
				_ = colour_primaries
				transfer_characteristics, err := r.ReadBits(8)
				if err != nil {
					return s, err
				}
				_ = transfer_characteristics
				matrix_coefficients, err := r.ReadBits(8)
				if err != nil {
					return s, err
				}
				_ = matrix_coefficients
			}
		}
		chroma_loc_info_present_flag, err := r.ReadBit()
		if err != nil {
			return s, err
		}
		if chroma_loc_info_present_flag != 0 {
			chroma_sample_loc_type_top_field, err := r.ReadSE()
			if err != nil {
				return s, err
			}
			_ = chroma_sample_loc_type_top_field

			chroma_sample_loc_type_bottom_field, err := r.ReadSE()
			if err != nil {
				return s, err
			}

			_ = chroma_sample_loc_type_bottom_field
		}

		timing_info_present_flag, err := r.ReadBit()
		if err != nil {
			return s, err
		}

		if timing_info_present_flag != 0 {
			num_units_in_tick, err := r.ReadBits(32)
			if err != nil {
				return s, err
			}
			time_scale, err := r.ReadBits(32)
			if err != nil {
				return s, err
			}
			s.FPS = int(math.Floor(float64(time_scale) / float64(num_units_in_tick) / 2.0))
			fixed_frame_rate_flag, err := r.ReadBit()
			if err != nil {
				return s, err
			}
			if fixed_frame_rate_flag != 0 {
				//utils.L.InfoLn("fixed_frame_rate_flag", fixed_frame_rate_flag)
				//have been devide 2
				//self.fps = self.fps / 2
			}
		}
	}
	return
}

func NewCodecDataFromAVCDecoderConfigurationRecord(record []byte) (*AVCDecoderConfigurationRecord, *SPS, error) {
	recordInfo := AVCDecoderConfigurationRecord{}
	if err := recordInfo.Unmarshal(record); err != nil {
		return nil, nil, err
	}

	spsInfo, err := ParseSPS(recordInfo.SPSList[0])
	if err != nil {
		return nil, nil, fmt.Errorf("h264parser: parse SPS failed(%s)", err)
	}
	return &recordInfo, &spsInfo, nil
}
