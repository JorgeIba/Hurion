package bootstrap

import (
	"errors"
	"fmt"

	"github.com/JorgeIba/Hurion/app/customerr/serviceerr"
	"github.com/JorgeIba/Hurion/app/logger"
	"github.com/JorgeIba/Hurion/app/services/service_model"
)

// ClassServiceInterface defines the subset of the ClassService needed for reconciliation.
//
//go:generate go tool mockgen -destination=mock_bootstrap/mock_reconciler.go -package=mock_bootstrap . ClassServiceInterface
type ClassServiceInterface interface {
	GetClassByClassCode(classCode string) (service_model.Class, error)
	CreateClass(actor service_model.Actor, classCode, className string) (int64, error)
	LinkClassroom(actor service_model.Actor, classroom service_model.Classroom) (bool, error)
	LinkClassAccessRole(actor service_model.Actor, role service_model.ClassAccessRole) (bool, error)
}

// Reconciler applies configuration using the same authorized services as runtime commands.
type Reconciler struct {
	classService ClassServiceInterface
	actor        service_model.Actor
}

func NewReconciler(classService ClassServiceInterface, actor service_model.Actor) *Reconciler {
	return &Reconciler{classService: classService, actor: actor}
}

// Reconcile performs additive conciliation based on the provided configuration.
func (r *Reconciler) Reconcile(cfg Config) error {
	logger.Log.Info("Starting configuration reconciliation...")

	for _, classCfg := range cfg.Classes {
		// 1. Check and Create Class
		_, err := r.classService.GetClassByClassCode(classCfg.Code)
		if err != nil {
			if _, ok := errors.AsType[*serviceerr.ClassCodeNotFoundError](err); ok {
				// Class does not exist, safe to create
				_, err = r.classService.CreateClass(r.actor, classCfg.Code, classCfg.Name)
				if err != nil {
					return fmt.Errorf("failed to create class %q: %w", classCfg.Code, err)
				}
				logger.Log.Infof("Created missing class %q.", classCfg.Code)
			} else {
				return fmt.Errorf("failed to check class %q: %w", classCfg.Code, err)
			}
		} else {
			logger.Log.Infof("Class %q already exists, skipping creation.", classCfg.Code)
		}

		for _, roomID := range classCfg.RoomIDs {
			linked, err := r.classService.LinkClassroom(r.actor, service_model.Classroom{
				ClassCode: classCfg.Code,
				RoomID:    roomID,
			})
			if err != nil {
				return fmt.Errorf("failed to link room %q to class %q: %w", roomID, classCfg.Code, err)
			}
			if linked {
				logger.Log.Infof("Linked room %q to class %q.", roomID, classCfg.Code)
			} else {
				logger.Log.Infof("Room %q already linked to class %q.", roomID, classCfg.Code)
			}
		}

		for _, roleID := range classCfg.RoleIDs {
			linked, err := r.classService.LinkClassAccessRole(r.actor, service_model.ClassAccessRole{
				ClassCode: classCfg.Code,
				RoleID:    roleID,
			})
			if err != nil {
				return fmt.Errorf("failed to link role %q to class %q: %w", roleID, classCfg.Code, err)
			}
			if linked {
				logger.Log.Infof("Linked role %q to class %q.", roleID, classCfg.Code)
			} else {
				logger.Log.Infof("Role %q already linked to class %q.", roleID, classCfg.Code)
			}
		}

	}

	logger.Log.Info("Configuration reconciliation complete.")
	return nil
}
