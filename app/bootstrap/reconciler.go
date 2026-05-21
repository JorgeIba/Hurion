package bootstrap

import (
	"errors"
	"fmt"

	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/customerr/serviceerr"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/logger"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/services/service_model"
)

// ClassServiceInterface defines the subset of the ClassService needed for reconciliation.
//
//go:generate go tool mockgen -destination=mock_bootstrap/mock_reconciler.go -package=mock_bootstrap . ClassServiceInterface
type ClassServiceInterface interface {
	GetClassByClassCode(classCode string) (service_model.Class, error)
	CreateClass(classCode, className string) (int64, error)

	GetClassCodeFromRoom(roomID string) (string, bool, error)
	RegisterClassroom(classroom service_model.Classroom) error

	HasAccessRole(classCode, roleID string) (bool, error)
	AssignAccessRole(role service_model.ClassAccessRole) error
}

// Reconciler synchronizes the database with the configuration file.
type Reconciler struct {
	classService ClassServiceInterface
}

func NewReconciler(classService ClassServiceInterface) *Reconciler {
	return &Reconciler{
		classService: classService,
	}
}

// Reconcile performs additive conciliation based on the provided configuration.
func (r *Reconciler) Reconcile(cfg Config) error {
	logger.Log.Info("Starting configuration reconciliation...")

	var notFoundErr *serviceerr.ClassCodeNotFoundError

	for _, classCfg := range cfg.Classes {
		// 1. Check and Create Class
		_, err := r.classService.GetClassByClassCode(classCfg.Code)
		if err != nil {
			if errors.As(err, &notFoundErr) {
				// Class does not exist, safe to create
				_, err = r.classService.CreateClass(classCfg.Code, classCfg.Name)
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

		// 2. Check and Link Rooms
		for _, roomID := range classCfg.RoomIDs {
			existingCode, exists, err := r.classService.GetClassCodeFromRoom(roomID)
			if err != nil {
				return fmt.Errorf("failed to check room %q: %w", roomID, err)
			}

			if exists {
				if existingCode == classCfg.Code {
					logger.Log.Infof("Room %q is already linked to class %q, skipping.", roomID, classCfg.Code)
					continue
				} else {
					return fmt.Errorf("room %q is already linked to a different class (%q)", roomID, existingCode)
				}
			}

			// Room doesn't exist, safe to register
			err = r.classService.RegisterClassroom(service_model.Classroom{
				ClassCode: classCfg.Code,
				RoomID:    roomID,
			})
			if err != nil {
				return fmt.Errorf("failed to link room %q to class %q: %w", roomID, classCfg.Code, err)
			}
			logger.Log.Infof("Ensured room %q is linked to class %q.", roomID, classCfg.Code)
		}

		// 3. Check and Link Roles
		for _, roleID := range classCfg.RoleIDs {
			exists, err := r.classService.HasAccessRole(classCfg.Code, roleID)
			if err != nil {
				return fmt.Errorf("failed to check role %q for class %q: %w", roleID, classCfg.Code, err)
			}

			if exists {
				logger.Log.Infof("Role %q is already linked to class %q, skipping.", roleID, classCfg.Code)
				continue
			}

			// Role is missing, safe to assign
			err = r.classService.AssignAccessRole(service_model.ClassAccessRole{
				ClassCode: classCfg.Code,
				RoleID:    roleID,
			})
			if err != nil {
				return fmt.Errorf("failed to link role %q to class %q: %w", roleID, classCfg.Code, err)
			}
			logger.Log.Infof("Ensured role %q is linked to class %q.", roleID, classCfg.Code)
		}
	}

	logger.Log.Info("Configuration reconciliation complete.")
	return nil
}
