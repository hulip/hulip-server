package db

import (
	"github.com/jinzhu/gorm"
)

type PlayState struct {
	gorm.Model
	UUID      string
	UserID    uint
	Finished  bool
	Playtime  float64
	OwnerID   uint
	OwnerType string
}

func CreatePlayState(userID uint, uuid string, finished bool, playtime float64) bool {
	var ps PlayState
	env.Db.FirstOrInit(&ps, PlayState{UUID: uuid, UserID: userID})
	ps.Finished = finished
	ps.Playtime = playtime
	env.Db.Save(&ps)

	count := 0
	var movie Movie
	var episode TvEpisode

	env.Db.Where("uuid = ?", uuid).Find(&movie).Count(&count)
	if count > 0 {
		movie.PlayState = ps
		env.Db.Save(&movie)
		return true
	}

	count = 0
	env.Db.Where("uuid = ?", uuid).Find(&episode).Count(&count)
	if count > 0 {
		episode.PlayState = ps
		env.Db.Save(&episode)
		return true
	}

	return false
}
