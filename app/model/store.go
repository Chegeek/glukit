package model

import (
	"app/util"
	"appengine/datastore"
	"fmt"
	"strconv"
	"time"
)

func (dayOfReads *DayOfGlucoseReads) Load(channel <-chan datastore.Property) error {
	var startTime time.Time
	var startTimeLocal string
	var userLocation *time.Location

	for property := range channel {
		switch columnName, columnValue := property.Name, property.Value; {
		case columnName == "startTime":
			startTime = columnValue.(time.Time)
		case columnName == "endTime":
			// We ignore it on load
			_ = columnValue.(time.Time)
		case columnName == "startTimeLocal":
			startTimeLocal = columnValue.(string)
			userLocation = util.GetLocaltimeOffset(startTimeLocal, startTime)
		default:
			offsetInSeconds, err := strconv.ParseInt(columnName, 10, 64)
			if err != nil {
				util.Propagate(err)
			}

			readTime := time.Unix(startTime.Unix()+offsetInSeconds, 0)
			// We need to convert value to int64 and cast it as int
			value := int(columnValue.(int64))

			localTime := readTime.In(userLocation).Format(util.TIMEFORMAT_NO_TZ)
			read := GlucoseRead{localTime, Timestamp(readTime.Unix()), value}
			dayOfReads.Reads = append(dayOfReads.Reads, read)
		}
	}

	return nil
}

func (dayOfReads *DayOfGlucoseReads) Save(channel chan<- datastore.Property) error {
	defer close(channel)

	size := len(dayOfReads.Reads)

	// Nothing to do if the slice has zero elements
	if size == 0 {
		return nil
	}
	reads := dayOfReads.Reads
	startTimestamp := int64(reads[0].Timestamp)
	startTime := time.Unix(startTimestamp, 0)
	startTimeLocal := reads[0].LocalTime
	endTimestamp := int64(reads[size-1].Timestamp)
	endTime := time.Unix(endTimestamp, 0)

	channel <- datastore.Property{
		Name:  "startTime",
		Value: startTime,
	}
	channel <- datastore.Property{
		Name:  "endTime",
		Value: endTime,
	}
	channel <- datastore.Property{
		Name:  "startTimeLocal",
		Value: startTimeLocal,
	}

	for i := range reads {
		readTimestamp := int64(reads[i].Timestamp)
		readOffset := readTimestamp - startTimestamp
		channel <- datastore.Property{
			Name:    strconv.FormatInt(readOffset, 10),
			Value:   int64(reads[i].Value),
			NoIndex: true,
		}
	}

	return nil
}

func (dayOfInjections *DayOfInjections) Load(channel <-chan datastore.Property) error {
	var startTime time.Time
	var startTimeLocal string
	var userLocation *time.Location

	for property := range channel {
		switch columnName, columnValue := property.Name, property.Value; {
		case columnName == "startTime":
			startTime = columnValue.(time.Time)
		case columnName == "endTime":
			// We ignore it on load
			_ = columnValue.(time.Time)
		case columnName == "startTimeLocal":
			startTimeLocal = columnValue.(string)
			userLocation = util.GetLocaltimeOffset(startTimeLocal, startTime)
		default:
			offsetInSeconds, err := strconv.ParseInt(columnName, 10, 64)
			if err != nil {
				util.Propagate(err)
			}

			timestamp := time.Unix(startTime.Unix()+offsetInSeconds, 0)
			// We need to convert value to float64 and we downcast to float32 (it's safe since we only up-casted it for
			// the store
			value := float32(columnValue.(float64))

			localTime := timestamp.In(userLocation).Format(util.TIMEFORMAT_NO_TZ)
			injection := Injection{localTime, Timestamp(timestamp.Unix()), value, UNDEFINED_READ}
			dayOfInjections.Injections = append(dayOfInjections.Injections, injection)
		}
	}

	return nil
}

func (dayOfInjections *DayOfInjections) Save(channel chan<- datastore.Property) error {
	defer close(channel)

	size := len(dayOfInjections.Injections)

	// Nothing to do if the slice has zero elements
	if size == 0 {
		return nil
	}
	injections := dayOfInjections.Injections
	startTimestamp := int64(injections[0].Timestamp)
	startTime := time.Unix(startTimestamp, 0)
	startTimeLocal := injections[0].LocalTime
	endTimestamp := int64(injections[size-1].Timestamp)
	endTime := time.Unix(endTimestamp, 0)

	channel <- datastore.Property{
		Name:  "startTime",
		Value: startTime,
	}
	channel <- datastore.Property{
		Name:  "endTime",
		Value: endTime,
	}
	channel <- datastore.Property{
		Name:  "startTimeLocal",
		Value: startTimeLocal,
	}

	for i := range injections {
		timestamp := int64(injections[i].Timestamp)
		offset := timestamp - startTimestamp
		// The datastore only supports float64 so we up-cast it to float64
		channel <- datastore.Property{
			Name:    strconv.FormatInt(offset, 10),
			Value:   float64(injections[i].Units),
			NoIndex: true,
		}
	}

	return nil
}

func (dayOfCarbs *DayOfCarbs) Load(channel <-chan datastore.Property) error {
	var startTime time.Time
	var startTimeLocal string
	var userLocation *time.Location

	for property := range channel {
		switch columnName, columnValue := property.Name, property.Value; {
		case columnName == "startTime":
			startTime = columnValue.(time.Time)
		case columnName == "endTime":
			// We ignore it on load
			_ = columnValue.(time.Time)
		case columnName == "startTimeLocal":
			startTimeLocal = columnValue.(string)
			userLocation = util.GetLocaltimeOffset(startTimeLocal, startTime)
		default:
			offsetInSeconds, err := strconv.ParseInt(columnName, 10, 64)
			if err != nil {
				util.Propagate(err)
			}

			timestamp := time.Unix(startTime.Unix()+offsetInSeconds, 0)
			// We need to convert value to float64 and we downcast to float32 (it's safe since we only up-casted it for
			// the store
			value := float32(columnValue.(float64))

			localTime := timestamp.In(userLocation).Format(util.TIMEFORMAT_NO_TZ)
			carb := Carb{localTime, Timestamp(timestamp.Unix()), value, UNDEFINED_READ}
			dayOfCarbs.Carbs = append(dayOfCarbs.Carbs, carb)
		}
	}

	return nil
}

func (dayOfCarbs *DayOfCarbs) Save(channel chan<- datastore.Property) error {
	defer close(channel)

	size := len(dayOfCarbs.Carbs)

	// Nothing to do if the slice has zero elements
	if size == 0 {
		return nil
	}
	carbs := dayOfCarbs.Carbs
	startTimestamp := int64(carbs[0].Timestamp)
	startTime := time.Unix(startTimestamp, 0)
	startTimeLocal := carbs[0].LocalTime
	endTimestamp := int64(carbs[size-1].Timestamp)
	endTime := time.Unix(endTimestamp, 0)

	channel <- datastore.Property{
		Name:  "startTime",
		Value: startTime,
	}
	channel <- datastore.Property{
		Name:  "endTime",
		Value: endTime,
	}
	channel <- datastore.Property{
		Name:  "startTimeLocal",
		Value: startTimeLocal,
	}

	for i := range carbs {
		timestamp := int64(carbs[i].Timestamp)
		offset := timestamp - startTimestamp
		// The datastore only supports float64 so we up-cast it to float64
		channel <- datastore.Property{
			Name:    strconv.FormatInt(offset, 10),
			Value:   float64(carbs[i].Grams),
			NoIndex: true,
		}
	}

	return nil
}

func (dayOfExercises *DayOfExercises) Load(channel <-chan datastore.Property) error {
	var startTime time.Time
	var startTimeLocal string
	var userLocation *time.Location

	for property := range channel {
		switch columnName, columnValue := property.Name, property.Value; {
		case columnName == "startTime":
			startTime = columnValue.(time.Time)
		case columnName == "endTime":
			// We ignore it on load
			_ = columnValue.(time.Time)
		case columnName == "startTimeLocal":
			startTimeLocal = columnValue.(string)
			userLocation = util.GetLocaltimeOffset(startTimeLocal, startTime)
		default:
			offsetInSeconds, err := strconv.ParseInt(columnName, 10, 64)
			if err != nil {
				util.Propagate(err)
			}

			timestamp := time.Unix(startTime.Unix()+offsetInSeconds, 0)
			// We split the value string to extract the two components from it
			value := columnValue.(string)
			var duration int
			var intensity string
			fmt.Sscanf(value, EXERCISE_VALUE_FORMAT, &duration, &intensity)

			localTime := timestamp.In(userLocation).Format(util.TIMEFORMAT_NO_TZ)
			exercise := Exercise{localTime, Timestamp(timestamp.Unix()), duration, intensity}
			dayOfExercises.Exercises = append(dayOfExercises.Exercises, exercise)
		}
	}

	return nil
}

func (dayOfExercises *DayOfExercises) Save(channel chan<- datastore.Property) error {
	defer close(channel)

	size := len(dayOfExercises.Exercises)

	// Nothing to do if the slice has zero elements
	if size == 0 {
		return nil
	}
	exercises := dayOfExercises.Exercises
	startTimestamp := int64(exercises[0].Timestamp)
	startTime := time.Unix(startTimestamp, 0)
	startTimeLocal := exercises[0].LocalTime
	endTimestamp := int64(exercises[size-1].Timestamp)
	endTime := time.Unix(endTimestamp, 0)

	channel <- datastore.Property{
		Name:  "startTime",
		Value: startTime,
	}
	channel <- datastore.Property{
		Name:  "endTime",
		Value: endTime,
	}
	channel <- datastore.Property{
		Name:  "startTimeLocal",
		Value: startTimeLocal,
	}

	for i := range exercises {
		timestamp := int64(exercises[i].Timestamp)
		offset := timestamp - startTimestamp
		// We need to store two values so we use a string and format each value inside of a single cell value
		channel <- datastore.Property{
			Name:    strconv.FormatInt(offset, 10),
			Value:   fmt.Sprintf(EXERCISE_VALUE_FORMAT, exercises[i].DurationInMinutes, exercises[i].Intensity),
			NoIndex: true,
		}
	}

	return nil
}