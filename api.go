package main

import (
	"encoding/json"
	"fmt"
	"github.com/alexandre-normand/glukit/app/apimodel"
	"github.com/alexandre-normand/glukit/app/bufio"
	"github.com/alexandre-normand/glukit/app/engine"
	"github.com/alexandre-normand/glukit/app/store"
	"github.com/alexandre-normand/glukit/app/streaming"
	"google.golang.org/appengine"
	"google.golang.org/appengine/log"
	"io"
	"net/http"
	"strings"
)

const (
	GLUCOSEREADS_V1_ROUTE = "v1_glucosereads"
	CALIBRATIONS_V1_ROUTE = "v1_calibrations"
	EXERCISES_V1_ROUTE    = "v1_exercises"
	MEALS_V1_ROUTE        = "v1_meals"
	INJECTIONS_V1_ROUTE   = "v1_injections"
)

// Represents the logging of a file import
type ApiUser struct {
	Email string
}

func CurrentApiUser(request *http.Request) (user *ApiUser) {
	request.ParseForm()

	authorizationValue := request.Header.Get("Authorization")
	if authorizationValue == "" {
		return nil
	}

	accessCode := strings.TrimPrefix(authorizationValue, "Bearer ")
	if accessCode == "" {
		return nil
	}

	// load access data
	if accessData, err := server.Storage.LoadAccess(accessCode, request); err == nil {
		return &ApiUser{accessData.UserData.(string)}
	}

	return nil
}

func initApiEndpoints(writer http.ResponseWriter, request *http.Request) {
	muxRouter.Get(CALIBRATIONS_V1_ROUTE).Handler(newOauthAuthenticationHandler(http.HandlerFunc(processNewCalibrationData)))
	muxRouter.Get(INJECTIONS_V1_ROUTE).Handler(newOauthAuthenticationHandler(http.HandlerFunc(processNewInjectionData)))
	muxRouter.Get(MEALS_V1_ROUTE).Handler(newOauthAuthenticationHandler(http.HandlerFunc(processNewMealData)))
	muxRouter.Get(GLUCOSEREADS_V1_ROUTE).Handler(newOauthAuthenticationHandler(http.HandlerFunc(processNewGlucoseReadData)))
	muxRouter.Get(EXERCISES_V1_ROUTE).Handler(newOauthAuthenticationHandler(http.HandlerFunc(processNewExerciseData)))
}

// processNewCalibrationData Handles a Post to the calibration endpoint and
// handles all data to be stored for a given user
func processNewCalibrationData(writer http.ResponseWriter, request *http.Request) {
	context := appengine.NewContext(request)
	user := CurrentApiUser(request)

	userProfileKey, _, err := store.GetGlukitUser(context, user.Email)
	if err != nil {
		log.Warningf(context, "Error getting user to process calibration data, user email is [%s]: %v", user.Email, err)
		http.Error(writer, "Error getting user to process calibration data", 500)
		return
	}

	dataStoreWriter := store.NewDataStoreCalibrationBatchWriter(context, userProfileKey)
	batchingWriter := bufio.NewCalibrationWriterSize(dataStoreWriter, store.GLUKIT_SCORE_PUT_MULTI_SIZE)
	calibrationStreamer := streaming.NewCalibrationReadStreamerDuration(batchingWriter, apimodel.DAY_OF_DATA_DURATION)

	decoder := json.NewDecoder(request.Body)

	for {
		var c []apimodel.CalibrationRead

		if err = decoder.Decode(&c); err == io.EOF {
			break
		} else if err != nil {
			log.Warningf(context, "Error processing calibration data for user [%s]: %v", user.Email, err)
			break
		}

		log.Debugf(context, "Writing new calibration reads [%v]", c)
		calibrationStreamer, err = calibrationStreamer.WriteCalibrations(c)
		if err != nil {
			log.Warningf(context, "Error storing calibration data [%v]: %v", c, err)
			http.Error(writer, fmt.Sprintf("Error storing data: %v", err), 502)
			return
		}
	}

	if err != io.EOF {
		log.Warningf(context, "Error processing calibration read data for user [%s]: %v", user.Email, err)
		http.Error(writer, fmt.Sprintf("Error decoding data: %v", err), 400)
		return
	}

	calibrationStreamer, err = calibrationStreamer.Close()
	if err != nil {
		log.Warningf(context, "Error closing calibration read streamer: %v", err)
		http.Error(writer, fmt.Sprintf("Error storing data: %v", err), 502)
		return
	}

	log.Infof(context, "Wrote calibrations to the datastore for user [%s]", user.Email)
	writer.WriteHeader(200)
}

// processNewGlucoseReadData Handles a Post to the glucosereads endpoint and
// handles all data to be stored for a given user
func processNewGlucoseReadData(writer http.ResponseWriter, request *http.Request) {
	context := appengine.NewContext(request)
	user := CurrentApiUser(request)

	userProfileKey, _, err := store.GetGlukitUser(context, user.Email)
	if err != nil {
		log.Warningf(context, "Error getting user to process glucose read data, user email is [%s]: %v", user.Email, err)
		http.Error(writer, "Error getting user to process glucose read data", 500)
		return
	}

	dataStoreWriter := store.NewDataStoreGlucoseReadBatchWriter(context, userProfileKey)
	batchingWriter := bufio.NewGlucoseReadWriterSize(dataStoreWriter, store.GLUKIT_SCORE_PUT_MULTI_SIZE)
	glucoseReadStreamer := streaming.NewGlucoseStreamerDuration(batchingWriter, apimodel.DAY_OF_DATA_DURATION)

	decoder := json.NewDecoder(request.Body)

	for {
		var c []apimodel.GlucoseRead

		if err = decoder.Decode(&c); err == io.EOF {
			break
		} else if err != nil {
			log.Warningf(context, "Error processing glucose read data for user [%s]: %v", user.Email, err)
			break
		}

		log.Debugf(context, "Writing [%d] new glucose reads: %v", len(c), c)
		glucoseReadStreamer, err = glucoseReadStreamer.WriteGlucoseReads(c)
		if err != nil {
			log.Warningf(context, "Error storing user data [%v]: %v", c, err)
			http.Error(writer, fmt.Sprintf("Error storing data: %v", err), 502)
			return
		}
	}

	if err != io.EOF {
		log.Warningf(context, "Error processing glucose read data for user [%s]: %v", user.Email, err)
		http.Error(writer, fmt.Sprintf("Error decoding data: %v", err), 400)
		return
	}

	glucoseReadStreamer, err = glucoseReadStreamer.Close()
	if err != nil {
		log.Warningf(context, "Error closing glucose read streamer: %v", err)
		http.Error(writer, fmt.Sprintf("Error storing data: %v", err), 502)
		return
	}

	_, glukitUser, err := store.GetGlukitUser(context, user.Email)
	if err != nil {
		log.Warningf(context, "Couldn't get glukit user profile [%s] to recalculate score: %v", user.Email, err)
	}

	err = engine.StartGlukitScoreBatch(context, glukitUser)
	if err != nil {
		log.Warningf(context, "Error starting glukit score calculation batch for user [%s]: %v", user.Email, err)
	}

	err = engine.StartA1CCalculationBatch(context, glukitUser)
	if err != nil {
		log.Warningf(context, "Error starting a1c calculation batch for user [%s]: %v", user.Email, err)
	}

	log.Infof(context, "Wrote glucose reads to the datastore for user [%s]", user.Email)
	writer.WriteHeader(200)
}

// processNewInjectionData Handles a Post to the injections endpoint and
// handles all data to be stored for a given user
func processNewInjectionData(writer http.ResponseWriter, request *http.Request) {
	context := appengine.NewContext(request)
	user := CurrentApiUser(request)

	userProfileKey, _, err := store.GetGlukitUser(context, user.Email)
	if err != nil {
		log.Warningf(context, "Error getting user to process injection data, user email is [%s]: %v", user.Email, err)
		http.Error(writer, "Error getting user to process injection data", 500)
		return
	}

	dataStoreWriter := store.NewDataStoreInjectionBatchWriter(context, userProfileKey)
	batchingWriter := bufio.NewInjectionWriterSize(dataStoreWriter, store.GLUKIT_SCORE_PUT_MULTI_SIZE)
	injectionStreamer := streaming.NewInjectionStreamerDuration(batchingWriter, apimodel.DAY_OF_DATA_DURATION)

	decoder := json.NewDecoder(request.Body)

	for {
		var p []apimodel.Injection

		if err = decoder.Decode(&p); err == io.EOF {
			break
		} else if err != nil {
			log.Warningf(context, "Error processing injection data for user [%s]: %v", user.Email, err)
			break
		}

		log.Debugf(context, "Writing [%d] new injections", len(p))
		injectionStreamer, err = injectionStreamer.WriteInjections(p)
		if err != nil {
			log.Warningf(context, "Error storing injection data [%v]: %v", p, err)
			http.Error(writer, fmt.Sprintf("Error storing data: %v", err), 502)
			return
		}
	}

	if err != io.EOF {
		log.Warningf(context, "Error processing injection data for user [%s]: %v", user.Email, err)
		http.Error(writer, fmt.Sprintf("Error decoding data: %v", err), 400)
		return
	}

	injectionStreamer, err = injectionStreamer.Close()
	if err != nil {
		log.Warningf(context, "Error closing injection streamer: %v", err)
		http.Error(writer, fmt.Sprintf("Error storing data: %v", err), 502)
		return
	}

	log.Infof(context, "Wrote injections to the datastore for user [%s]", user.Email)
	writer.WriteHeader(200)
}

// processNewMealData Handles a Post to the Meals endpoint and
// handles all data to be stored for a given user
func processNewMealData(writer http.ResponseWriter, request *http.Request) {
	context := appengine.NewContext(request)
	user := CurrentApiUser(request)

	userProfileKey, _, err := store.GetGlukitUser(context, user.Email)
	if err != nil {
		log.Warningf(context, "Error getting user to process meal data, user email is [%s]: %v", user.Email, err)
		http.Error(writer, "Error getting user to process meal data", 500)
		return
	}

	dataStoreWriter := store.NewDataStoreMealBatchWriter(context, userProfileKey)
	batchingWriter := bufio.NewMealWriterSize(dataStoreWriter, store.GLUKIT_SCORE_PUT_MULTI_SIZE)
	mealStreamer := streaming.NewMealStreamerDuration(batchingWriter, apimodel.DAY_OF_DATA_DURATION)

	decoder := json.NewDecoder(request.Body)

	for {
		var meals []apimodel.Meal

		if err = decoder.Decode(&meals); err == io.EOF {
			break
		} else if err != nil {
			log.Warningf(context, "Error processing meal data for user [%s]: %v", user.Email, err)
			break
		}

		log.Debugf(context, "Writing [%d] new meals", len(meals))
		mealStreamer, err = mealStreamer.WriteMeals(meals)
		if err != nil {
			log.Warningf(context, "Error storing meal data [%v]: %v", meals, err)
			http.Error(writer, fmt.Sprintf("Error storing meal data: %v", err), 502)
			return
		}
	}

	if err != io.EOF {
		log.Warningf(context, "Error processing meal data for user [%s]: %v", user.Email, err)
		http.Error(writer, fmt.Sprintf("Error decoding data: %v", err), 400)
		return
	}

	mealStreamer, err = mealStreamer.Close()
	if err != nil {
		log.Warningf(context, "Error closing meal streamer: %v", err)
		http.Error(writer, fmt.Sprintf("Error storing data: %v", err), 502)
		return
	}

	log.Infof(context, "Wrote meals to the datastore for user [%s]", user.Email)
	writer.WriteHeader(200)
}

// processNewExerciseData Handles a Post to the exercises endpoint and
// handles all data to be stored for a given user
func processNewExerciseData(writer http.ResponseWriter, request *http.Request) {
	context := appengine.NewContext(request)
	user := CurrentApiUser(request)

	userProfileKey, _, err := store.GetGlukitUser(context, user.Email)
	if err != nil {
		log.Warningf(context, "Error getting user to process exercise data, user email is [%s]: %v", user.Email, err)
		http.Error(writer, "Error getting user to process exercise data", 500)
		return
	}

	dataStoreWriter := store.NewDataStoreExerciseBatchWriter(context, userProfileKey)
	batchingWriter := bufio.NewExerciseWriterSize(dataStoreWriter, store.GLUKIT_SCORE_PUT_MULTI_SIZE)
	exerciseStreamer := streaming.NewExerciseStreamerDuration(batchingWriter, apimodel.DAY_OF_DATA_DURATION)

	decoder := json.NewDecoder(request.Body)

	for {
		var exercises []apimodel.Exercise

		if err = decoder.Decode(&exercises); err == io.EOF {
			break
		} else if err != nil {
			log.Warningf(context, "Error processing exercise data for user [%s]: %v", user.Email, err)
			break
		}

		log.Debugf(context, "Writing [%d] new Exercises", len(exercises))
		exerciseStreamer, err = exerciseStreamer.WriteExercises(exercises)
		if err != nil {
			log.Warningf(context, "Error storing exercise data [%v]: %v", exercises, err)
			http.Error(writer, fmt.Sprintf("Error storing exercise data: %v", err), 502)
			return
		}
	}

	if err != io.EOF {
		log.Warningf(context, "Error processing exercise data for user [%s]: %v", user.Email, err)
		http.Error(writer, fmt.Sprintf("Error decoding data: %v", err), 400)
		return
	}

	exerciseStreamer, err = exerciseStreamer.Close()
	if err != nil {
		log.Warningf(context, "Error closing exercise streamer: %v", err)
		http.Error(writer, fmt.Sprintf("Error storing data: %v", err), 502)
		return
	}

	log.Infof(context, "Wrote exercises to the datastore for user [%s]", user.Email)
	writer.WriteHeader(200)
}
