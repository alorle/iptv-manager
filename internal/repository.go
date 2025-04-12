package domain

type ChannelRepository interface {
	GetAll() ([]*Channel, error)
}
