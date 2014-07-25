package db

import (
	"errors"
	"fmt"
	"log"
	"pilosa/config"
	"pilosa/util"
	"sync"

	"github.com/stathat/consistent"
)

var FrameDoesNotExistError = errors.New("Frame does not exist.")
var InvalidFrameError = errors.New("Invalid frame.")
var SliceDoesNotExistError = errors.New("Slice does not exist.")
var FragmentDoesNotExistError = errors.New("Fragment does not exist.")
var FrameSliceIntersectDoesNotExistError = errors.New("FrameSliceIntersect does not exist.")

type Location struct {
	ProcessId  *util.GUID
	FragmentId util.SUUID
}

type Process struct {
	id        *util.GUID
	host      string
	port_tcp  int
	port_http int
	mutex     sync.Mutex
}

func NewProcess(id *util.GUID) *Process {
	return &Process{id: id}
}

func (self *Process) Id() util.GUID {
	self.mutex.Lock()
	defer self.mutex.Unlock()
	return *self.id
}

func (self *Process) Host() string {
	self.mutex.Lock()
	defer self.mutex.Unlock()
	return self.host
}

func (self *Process) SetHost(host string) {
	self.mutex.Lock()
	defer self.mutex.Unlock()
	self.host = host
}

func (self *Process) PortTcp() int {
	self.mutex.Lock()
	defer self.mutex.Unlock()
	return self.port_tcp
}

func (self *Process) SetPortTcp(port int) {
	self.mutex.Lock()
	defer self.mutex.Unlock()
	self.port_tcp = port
}

func (self *Process) PortHttp() int {
	self.mutex.Lock()
	defer self.mutex.Unlock()
	return self.port_http
}

func (self *Process) SetPortHttp(port int) {
	self.mutex.Lock()
	defer self.mutex.Unlock()
	self.port_http = port
}

/*
// Create a Location struct given a string in form "0.0.0.0:0"
func NewLocation(location_string string) (*Location, error) {
	splitstring := strings.Split(location_string, ":")
	if len(splitstring) != 2 {
		return nil, errors.New("Location string must be in form 0.0.0.0:0")
	}
	ip := splitstring[0]
	port, err := strconv.Atoi(splitstring[1])
	if err != nil {
		return nil, errors.New("Port is not a number!")
	}
	return &Location{ip, port}, nil
}

func (location *Location) ToString() string {
	return fmt.Sprintf("%s:%d", location.Ip, location.Port)
}

// Map of node location to their router
type NodeMap map[Location]Location
*/

/////////// CLUSTERS ////////////////////////////////////////////////////////////////////

// Represents the entire cluster, and a reference to the Node this instance is running on
type Cluster struct {
	databases map[string]*Database
	mutex     sync.Mutex
}

func NewCluster() *Cluster {
	cluster := Cluster{}
	cluster.databases = make(map[string]*Database)
	return &cluster
}
func (self *Cluster) GetDatabases() map[string]*Database {
	return self.databases

}

/////////// DATABASES ////////////////////////////////////////////////////////////////////

// A database is a collection of all the frames within a given profile space
type Database struct {
	Name                   string
	frames                 []*Frame
	slices                 []*Slice
	frame_slice_intersects []*FrameSliceIntersect
	mutex                  sync.Mutex
}

func (self *Database) GetFrameSliceIntersects() []*FrameSliceIntersect {
	return self.frame_slice_intersects
}

// Add a database to a cluster
func (c *Cluster) addDatabase(name string) *Database {
	database := Database{Name: name}
	if c.databases == nil {
		c.databases = make(map[string]*Database)
	}
	c.databases[name] = &database
	return &database
}

func (c *Cluster) getDatabase(name string) (*Database, error) {
	value, ok := c.databases[name]
	if !ok {
		return nil, errors.New("The database does not exist!")
	} else {
		return value, nil
	}
}

func (c *Cluster) GetOrCreateDatabase(name string) *Database {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	database, err := c.getDatabase(name)
	if err == nil {
		return database
	}
	return c.addDatabase(name)
}

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func (d *Database) IsValidFrame(name string) bool {
	supported_frames := config.GetStringArrayDefault("supported_frames", []string{"default"})
	return stringInSlice(name, supported_frames)
}

// Count the number of slices in a database
func (d *Database) NumSlices() (int, error) {
	if len(d.slices) < 1 {
		return 0, errors.New("Database is empty")
	}
	return len(d.slices), nil
}

///////// FRAMES ////////////////////////////////////////////////////////////////////

// A frame is a collection of slices in a given category
// (brands, demographics, etc), specific to a database
type Frame struct {
	name string
}

// Get a frame from a database
func (d *Database) getFrame(name string) (*Frame, error) {
	// should we check here for supported frames?
	if !d.IsValidFrame(name) {
		return nil, InvalidFrameError
	}
	for _, frame := range d.frames {
		if frame.name == name {
			return frame, nil
		}
	}
	return nil, FrameDoesNotExistError
}

// Add a frame to a database
func (d *Database) addFrame(name string) *Frame {
	frame := Frame{name: name}
	d.frames = append(d.frames, &frame)
	// add intersections
	for _, slice := range d.slices {
		d.AddFrameSliceIntersect(&frame, slice)
	}
	return &frame
}

func (d *Database) GetOrCreateFrame(name string) *Frame {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	frame, err := d.getFrame(name)
	if err == nil {
		return frame
	}
	return d.addFrame(name)
}

///////// SLICES /////////////////////////////////////////////////////////////////////////

// A slice is the vertical combination of every fragment.
type Slice struct {
	id int
}

func (self *Slice) Id() int {
	return self.id
}

// Get a slice from a database
func (d *Database) getSlice(slice_id int) (*Slice, error) {
	for _, slice := range d.slices {
		if slice.id == slice_id {
			return slice, nil
		}
	}
	return nil, SliceDoesNotExistError
}

// Add a slice to a database
func (d *Database) addSlice(slice_id int) *Slice {
	slice := Slice{id: slice_id}
	d.slices = append(d.slices, &slice)
	// add intersections
	for _, frame := range d.frames {
		d.AddFrameSliceIntersect(frame, &slice)
	}
	return &slice
}

func (d *Database) GetOrCreateSlice(slice_id int) *Slice {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	slice, err := d.getSlice(slice_id)
	if err == nil {
		return slice
	}
	return d.addSlice(slice_id)
}

///////// FRAME-SLICE INTERSECT //////////////////////////////////////////////////////////////

type FrameSliceIntersect struct {
	frame     *Frame
	slice     *Slice
	fragments []*Fragment
	hashring  *consistent.Consistent
}

func (d *Database) AddFrameSliceIntersect(frame *Frame, slice *Slice) *FrameSliceIntersect {
	frameslice := FrameSliceIntersect{frame: frame, slice: slice}
	d.frame_slice_intersects = append(d.frame_slice_intersects, &frameslice)
	frameslice.hashring = consistent.New()
	frameslice.hashring.NumberOfReplicas = 16
	return &frameslice
}

func (d *Database) GetFrameSliceIntersect(frame *Frame, slice *Slice) (*FrameSliceIntersect, error) {
	for _, frameslice := range d.frame_slice_intersects {
		if frameslice.frame == frame && frameslice.slice == slice {
			return frameslice, nil
		}
	}
	log.Println("Missing FrameSliceIntersect:", d.Name, frame, slice)
	return nil, FrameSliceIntersectDoesNotExistError
}

func (self *FrameSliceIntersect) GetFragments() []*Fragment {
	return self.fragments
}

func (self *FrameSliceIntersect) GetFragment(fragment_id util.SUUID) (*Fragment, error) {
	for _, fragment := range self.fragments {
		if fragment.id == fragment_id {
			return fragment, nil
		}
	}
	return nil, FragmentDoesNotExistError
}

func (self *FrameSliceIntersect) AddFragment(fragment *Fragment) {
	self.fragments = append(self.fragments, fragment)
	self.hashring.Add(util.SUUID_to_Hex(fragment.id))
}

///////// FRAGMENTS ////////////////////////////////////////////////////////////////////////

// A fragment is a collection of bitmaps within a slice. The fragment contains a reference to the responsible node for that fragment. The node is in the form ip:port
type Fragment struct {
	id      util.SUUID
	process *Process
}

func (self *Fragment) GetId() util.SUUID {
	return self.id
}

func (self *Fragment) GetProcess() *Process {
	return self.process
}

func (self *Fragment) GetProcessId() *util.GUID {
	return self.process.id
}

func (self *Fragment) GetLocation() *Location {
	return &Location{self.process.id, self.id}
}

// rename this one
func (d *Database) GetFragmentForBitmap(slice *Slice, bitmap *Bitmap) (*Fragment, error) {
	//d.mutex.Lock()
	//defer d.mutex.Unlock()
	frame, err := d.getFrame(bitmap.FrameType)
	if err != nil {
		log.Println("Missing FrameType", bitmap.FrameType, d.Name, slice)
		log.Println(err)
		return nil, err
	}
	//log.Println(frame, slice)
	fsi, err := d.GetFrameSliceIntersect(frame, slice)
	if err != nil {
		log.Println("Missing frame,slice", frame, slice)
		log.Println(err)
		return nil, err
	}
	frag_id_s, err := fsi.hashring.Get(fmt.Sprintf("%d", bitmap.Id))
	if err != nil {
		log.Println("ERROR FSI.GET:", bitmap.Id, bitmap.FrameType, d.Name, frame, slice)
		log.Println(err)
		return nil, err
	}
	frag_id := util.Hex_to_SUUID(frag_id_s)
	return fsi.GetFragment(frag_id)
}

func (d *Database) GetFragmentForFrameSlice(frame *Frame, slice *Slice) (*Fragment, error) {
	fsi, err := d.GetFrameSliceIntersect(frame, slice)
	if err != nil {
		log.Println("Missing frame,slice", frame, slice)
		log.Println(err)
		return nil, err
	}
	frag_id_s, err := fsi.hashring.Get("0") // we don't need a specific bitmap in here because we're assuming the hashring only has a single element
	if err != nil {
		log.Println("ERROR FSI.GET:", d.Name, frame, slice)
		log.Println(err)
		return nil, err
	}
	frag_id := util.Hex_to_SUUID(frag_id_s)
	return fsi.GetFragment(frag_id)
}

/*
// NOT IMPLEMENTED
// this would loop through all frame_slice_intersect[], then all fragmments to find a match
func (d *Database) GetFragmentById(fragment_id *GUID) *Fragment {
}
*/

func (d *Database) getFragment(frame *Frame, slice *Slice, fragment_id util.SUUID) (*Fragment, error) {
	fsi, err := d.GetFrameSliceIntersect(frame, slice)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	return fsi.GetFragment(fragment_id)
}

func (d *Database) addFragment(frame *Frame, slice *Slice, fragment_id util.SUUID) *Fragment {
	fsi, err := d.GetFrameSliceIntersect(frame, slice)
	if err != nil {
		log.Println("database.addFragment", err)
		return nil
	}
	fragment := Fragment{id: fragment_id}
	fsi.AddFragment(&fragment)
	return &fragment
}

/*
func (d *Database) AllocateFragment(frame *Frame, slice *Slice) *Fragment {
    // from ETCD, randomly get a process that has available_fragments > 0
    // atomically decrement available_fragments (as long as it's not 0)
    // if it IS 0, try until we find a process with available capacity

    *
    process, err := GetAvailableProcess()
	if err != nil {
		log.Fatal(err)
	}
    *
    process_id, _ := uuid.NewV4()
    process := NewProcess(process_id)
    return nil
    //return d.AddFragment(&frame, &slice, process)
}
*/

/*
func (d *Database) AddFragmentByProcess(frame *Frame, slice *Slice, process *Process) *Fragment {
    d.mutex.Lock()
    defer d.mutex.Unlock()
	frameslice, _ := d.GetFrameSliceIntersect(frame, slice)
    fragment_id, _ := uuid.NewV4()
    fragment := Fragment{id: fragment_id, process: process}
    frameslice.fragments = append(frameslice.fragments, &fragment)
    frameslice.hashring.Add(fragment.id.String())
    return &fragment
}
*/

func (d *Database) GetOrCreateFragment(frame *Frame, slice *Slice, fragment_id util.SUUID) *Fragment {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	fragment, err := d.getFragment(frame, slice, fragment_id)
	if err == nil {
		return fragment
	}
	return d.addFragment(frame, slice, fragment_id)
}

func (f *Fragment) SetProcess(process *Process) {
	f.process = process
}
func GetSlice(profile_id uint64) int {
	return int(profile_id / SLICE_WIDTH)
}

///////////////////////////////////////////////////////////////////////////////////////////////

// Get a slice from a database

func (d *Database) GetSliceForProfile(profile_id uint64) (*Slice, error) {
	return d.getSlice(GetSlice(profile_id))
}

type Bitmap struct {
	Id        uint64
	FrameType string
	Filter    uint64
}
