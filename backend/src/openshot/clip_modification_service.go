package openshot

import (
	"github.com/mitchellh/mapstructure"
)

const (
	LocationX = "location_x"
	// http://cloud.openshot.org/doc/animation.html#key-frames
	// 0=Bézier, 1=Linear, 2=Constant
	interpolationMode = 1
)

// SetScale sets the scale of the provided clip object
// DOES NOT set value on server, requires call to UpdateClip
func (o *OpenShot) SetScale(clip *Clip, scale int) {
	clip.JSON["scale"] = scale
}

// AddPropertyPoint sets a JSON property of the provided clip object at the specified frame
// DOES NOT set value on server, requires call to UpdateClip
func (o *OpenShot) AddPropertyPoint(clip *Clip, key string, frame int, value float64) {
	property := o.GetProperty(clip, key)
	property.Points = append(property.Points, Point{Co: Cord{X: frame, Y: value}, Interpolation: interpolationMode})
	clip.JSON[key] = property
}

// GetProperty returns a json object type-asserted to an openshot.Property object
func (o *OpenShot) GetProperty(clip *Clip, key string) *Property {
	var property Property
	mapstructure.Decode(clip.JSON[key], &property)
	return &property
}
