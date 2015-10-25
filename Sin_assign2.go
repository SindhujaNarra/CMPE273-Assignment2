package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/julienschmidt/httprouter"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type MongoSession struct {
	session *mgo.Session
}

func newMongoSession(session *mgo.Session) *MongoSession {
	return &MongoSession{session}
}

/*
var (
	session    *mgo.Session
	collection *mgo.Collection
)
*/
type Coordinates struct {
	Results []struct {
		AddressComponents []struct {
			LongName  string   `json:"long_name"`
			ShortName string   `json:"short_name"`
			Types     []string `json:"types"`
		} `json:"address_components"`
		FormattedAddress string `json:"formatted_address"`
		Geometry         struct {
			Location struct {
				Long float64 `json:"long"`
				Lat  float64 `json:"lat"`
			} `json:"location"`
			LocationType string `json:"location_type"`
			Viewport     struct {
				Northeast struct {
					Long float64 `json:"long"`
					Lat  float64 `json:"lat"`
				} `json:"northeast"`
				Southwest struct {
					Long float64 `json:"long"`
					Lat  float64 `json:"lat"`
				} `json:"southwest"`
			} `json:"viewport"`
		} `json:"geometry"`
		PlaceID string   `json:"place_id"`
		Types   []string `json:"types"`
	} `json:"results"`
	Status string `json:"status"`
}

type Response struct {
	ID         bson.ObjectId `bson:"_id" json:"id"`
	Name       string        `json:"name"`
	Address    string        `json:"address"`
	City       string        `json:"city"`
	State      string        `json:"state"`
	Zip        int           `bson:"zip" json:"zip"`
	Coordinate struct {
		Long float64 `json:"long"   bson:"long"`
		Lat  float64 `json:"lat"   bson:"lat"`
	} `json:"coordinate" bson:"coordinate"`
}

// calling google maps api for location
func callGoogleAPI(res *Response) Response {

	address := res.Address
	city := res.City

	gstate := strings.Replace(res.State, " ", "+", -1)
	gaddress := strings.Replace(address, " ", "+", -1)
	gcity := strings.Replace(city, " ", "+", -1)

	uri := "http://maps.google.com/maps/api/geocode/json?address=" + gaddress + "+" + gcity + "+" + gstate + "&sensor=false"

	result, _ := http.Get(uri)

	body, _ := ioutil.ReadAll(result.Body)

	Coords := Coordinates{}

	err := json.Unmarshal(body, &Coords)
	if err != nil {
		panic(err)
	}

	for _, Sample := range Coords.Results {
		res.Coordinate.Long = Sample.Geometry.Location.Long
		res.Coordinate.Lat = Sample.Geometry.Location.Lat

	}

	return *res
}

// Get the details
func (ms MongoSession) HandleGetLocation(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	id := p.ByName("id")
	if !bson.IsObjectIdHex(id) {
		w.WriteHeader(404)
		return
	}

	fmt.Print("Before ObjectID")
	oid := bson.ObjectIdHex(id)
	fmt.Print("New ObjectID is", oid)

	res := Response{}

	if err := ms.session.DB("sindb").C("locations").FindId(oid).One(&res); err != nil {
		fmt.Print("Fail case")
		w.WriteHeader(404)
		return
	}

	json.NewDecoder(r.Body).Decode(res)

	outJSON, _ := json.Marshal(res)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	fmt.Fprintf(w, "%s", outJSON)
}

// creating and stroing the details
func (ms MongoSession) HandlePostLocation(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	res := Response{}

	fmt.Println("vvmk")
	json.NewDecoder(r.Body).Decode(&res)
	fmt.Println("jksahduhf")
	data := callGoogleAPI(&res)

	data.ID = bson.NewObjectId()

	ms.session.DB("sindb").C("locations").Insert(data)

	outJSON, _ := json.Marshal(data)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(201)
	fmt.Fprintf(w, "%s", outJSON)

}

// Deleting the particular id details
func (ms MongoSession) HandleDeleteLocation(w http.ResponseWriter, r *http.Request, p httprouter.Params) {

	id := p.ByName("id")

	if !bson.IsObjectIdHex(id) {
		w.WriteHeader(404)
		return
	}

	oid := bson.ObjectIdHex(id)
	if err := ms.session.DB("sindb").C("locations").RemoveId(oid); err != nil {
		fmt.Print("Inside fail case")
		w.WriteHeader(404)
		return
	}

	w.WriteHeader(200)
}

// Updating particular id details
func (ms MongoSession) HandlePutLocation(w http.ResponseWriter, r *http.Request, p httprouter.Params) {

	id := p.ByName("id")

	if !bson.IsObjectIdHex(id) {
		w.WriteHeader(404)
		return
	}

	oid := bson.ObjectIdHex(id)

	get := Response{}
	put := Response{}

	put.ID = oid

	json.NewDecoder(r.Body).Decode(&put)

	if err := ms.session.DB("sindb").C("locations").FindId(oid).One(&get); err != nil {

		w.WriteHeader(404)
		return
	}

	na := get.Name

	object := ms.session.DB("sindb").C("locations")

	get = callGoogleAPI(&put)
	object.Update(bson.M{"_id": oid}, bson.M{"$set": bson.M{"address": put.Address, "city": put.City, "state": put.State, "zip": put.Zip, "coordinate": bson.M{"lat": get.Coordinate.Lat, "long": get.Coordinate.Long}}})

	get.Name = na

	outJSON, _ := json.Marshal(get)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(201)
	fmt.Fprintf(w, "%s", outJSON)
}

// Starting and connecting with mongolab
func getConnection() *mgo.Session {

	fmt.Println("Starting Mongodb session")
	conn, err := mgo.Dial("mongodb://sindhuja:sindhuja@ds043694.mongolab.com:43694/sindb")

	if err != nil {
		panic(err)
	}
	return conn
}

func main() {

	r := httprouter.New()

	ms := newMongoSession(getConnection())

	r.GET("/locations/:id", ms.HandleGetLocation)
	r.POST("/locations", ms.HandlePostLocation)
	r.DELETE("/locations/:id", ms.HandleDeleteLocation)
	r.PUT("/locations/:id", ms.HandlePutLocation)

	fmt.Println("Listening on 8080")
	http.ListenAndServe("localhost:8080", r)

}
