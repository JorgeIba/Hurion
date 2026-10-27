package services

import "github.com/JorgeIba/Hurion/app/services/service_model"

func UserActor(userID string) service_model.Actor {
	return service_model.Actor{ID: userID, Kind: service_model.ActorUser}
}

// ConfigReconcilerActor identifies the trusted startup configuration reconciler.
func ConfigReconcilerActor() service_model.Actor {
	return service_model.Actor{ID: "config-reconciler", Kind: service_model.ActorProcess}
}
