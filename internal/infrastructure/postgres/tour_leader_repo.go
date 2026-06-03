package postgres

import (
	"context"
	"database/sql"

	"github.com/irfan-ghzl/pintour-travel/internal/domain/user"
)

type tourLeaderRepo struct{ db *sql.DB }

func NewTourLeaderRepo(db *sql.DB) user.TourLeaderRepository { return &tourLeaderRepo{db} }

func (r *tourLeaderRepo) Create(ctx context.Context, tl *user.TourLeader) error {
	return r.db.QueryRowContext(ctx, `
		INSERT INTO tour_leaders(id,user_id,bio,photo_path,experience_years,specialization,emergency_phone,created_at)
		VALUES(gen_random_uuid(),$1,$2,$3,$4,$5,$6,NOW())
		RETURNING id,created_at`,
		tl.UserID, tl.Bio, tl.PhotoPath, tl.ExperienceYears, tl.Specialization, tl.EmergencyPhone,
	).Scan(&tl.ID, &tl.CreatedAt)
}

func (r *tourLeaderRepo) GetByUserID(ctx context.Context, userID string) (*user.TourLeader, error) {
	var tl user.TourLeader
	err := r.db.QueryRowContext(ctx, `
		SELECT tl.id,tl.user_id,COALESCE(tl.bio,''),COALESCE(tl.photo_path,''),
		tl.experience_years,COALESCE(tl.specialization,''),COALESCE(tl.emergency_phone,''),
		tl.created_at,u.name,u.email,COALESCE(u.phone,'')
		FROM tour_leaders tl JOIN users u ON u.id=tl.user_id
		WHERE tl.user_id=$1`, userID,
	).Scan(&tl.ID, &tl.UserID, &tl.Bio, &tl.PhotoPath, &tl.ExperienceYears,
		&tl.Specialization, &tl.EmergencyPhone, &tl.CreatedAt,
		&tl.Name, &tl.Email, &tl.Phone)
	if err != nil {
		return nil, err
	}
	return &tl, nil
}

func (r *tourLeaderRepo) Update(ctx context.Context, tl *user.TourLeader) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE tour_leaders SET bio=$1,photo_path=$2,experience_years=$3,
		specialization=$4,emergency_phone=$5
		WHERE user_id=$6`,
		tl.Bio, tl.PhotoPath, tl.ExperienceYears,
		tl.Specialization, tl.EmergencyPhone, tl.UserID)
	return err
}

func (r *tourLeaderRepo) List(ctx context.Context) ([]user.TourLeader, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT tl.id,tl.user_id,COALESCE(tl.bio,''),COALESCE(tl.photo_path,''),
		tl.experience_years,COALESCE(tl.specialization,''),COALESCE(tl.emergency_phone,''),
		tl.created_at,u.name,u.email,COALESCE(u.phone,'')
		FROM tour_leaders tl JOIN users u ON u.id=tl.user_id
		ORDER BY u.name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []user.TourLeader
	for rows.Next() {
		var tl user.TourLeader
		if err := rows.Scan(&tl.ID, &tl.UserID, &tl.Bio, &tl.PhotoPath,
			&tl.ExperienceYears, &tl.Specialization, &tl.EmergencyPhone,
			&tl.CreatedAt, &tl.Name, &tl.Email, &tl.Phone); err != nil {
			return nil, err
		}
		list = append(list, tl)
	}
	return list, rows.Err()
}
