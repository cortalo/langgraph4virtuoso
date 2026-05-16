package canvas

type TrackSegmentStorageImpl struct {
	M3TrackWidth int
	Tracks       []Track
}

func NewTrackSegmentStorage(trackCount, trackWidth int) *TrackSegmentStorageImpl {
	tracks := make([]Track, trackCount)
	for i := range tracks {
		tracks[i] = NewTrackImpl()
	}
	return &TrackSegmentStorageImpl{
		M3TrackWidth: trackWidth,
		Tracks:       tracks,
	}
}

func (tss *TrackSegmentStorageImpl) IsPassible(seg TrackSegment) bool {
	if seg.TrackID < 0 || seg.TrackID >= len(tss.Tracks) {
		return false
	}
	return tss.Tracks[seg.TrackID].IsPassible(seg.NetID, seg.Start, seg.End)
}

func (tss *TrackSegmentStorageImpl) Occupy(seg TrackSegment) error {
	if seg.TrackID < 0 || seg.TrackID >= len(tss.Tracks) {
		return ErrInvalidTrackID
	}
	return tss.Tracks[seg.TrackID].Occupy(seg.NetID, seg.Start, seg.End)
}

func (tss *TrackSegmentStorageImpl) GetM3TrackWidth() int {
	return tss.M3TrackWidth
}
