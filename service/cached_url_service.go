package service

import (
	"context"
	"errors"
	"log"

	"goshort/cache"
	"goshort/model"
	"goshort/repository"
)

type CachedURLService struct {
	repository repository.URLRepository
	cache      *cache.RedisCache
}

func NewCachedURLService(
	repository repository.URLRepository,
	redisCache *cache.RedisCache,
) *CachedURLService {
	return &CachedURLService{
		repository: repository,
		cache:      redisCache,
	}
}

func (s *CachedURLService) CreateShortURL(
	ctx context.Context,
	originalURL string,
	userID int64,
) (model.URL, error) {
	baseService := NewURLService(s.repository)

	return baseService.CreateShortURL(
		ctx,
		originalURL,
		userID,
	)
}

func (s *CachedURLService) CreateShortURLWithAlias(
	ctx context.Context,
	originalURL string,
	alias string,
	userID int64,
) (model.URL, error) {
	baseService := NewURLService(s.repository)

	return baseService.CreateShortURLWithAlias(
		ctx,
		originalURL,
		alias,
		userID,
	)
}

func (s *CachedURLService) GetURLByID(
	ctx context.Context,
	id int64,
) (model.URL, error) {
	return s.repository.FindByID(ctx, id)
}

func (s *CachedURLService) ListURLs(
	ctx context.Context,
	userID int64,
) ([]model.URL, error) {
	baseService := NewURLService(s.repository)
	return baseService.ListURLs(ctx, userID)
}

func (s *CachedURLService) GetOriginalURL(
	ctx context.Context,
	shortCode string,
) (model.URL, error) {
	if s.cache != nil {
		cachedURL, err := s.cache.GetURL(ctx, shortCode)

		if err == nil {
			return cachedURL, nil
		}

		if !errors.Is(err, cache.ErrCacheMiss) {
			log.Printf(
				"redis cache read failed for %q: %v",
				shortCode,
				err,
			)
		}
	}

	url, err := s.repository.FindByShortCode(
		ctx,
		shortCode,
	)
	if err != nil {
		return model.URL{}, err
	}

	if s.cache != nil {
		if err := s.cache.SetURL(ctx, url); err != nil {
			log.Printf(
				"redis cache write failed for %q: %v",
				shortCode,
				err,
			)
		}
	}

	return url, nil
}

func (s *CachedURLService) UpdateURL(
	ctx context.Context,
	url model.URL,
) (model.URL, error) {
	baseService := NewURLService(s.repository)

	updatedURL, err := baseService.UpdateURL(
		ctx,
		url,
	)
	if err != nil {
		return model.URL{}, err
	}

	if s.cache != nil {
		if err := s.cache.DeleteURL(
			ctx,
			updatedURL.ShortCode,
		); err != nil {
			log.Printf(
				"redis cache invalidation failed for %q: %v",
				updatedURL.ShortCode,
				err,
			)
		}
	}

	return updatedURL, nil
}

func (s *CachedURLService) DeleteURL(
	ctx context.Context,
	id int64,
) error {
	url, err := s.repository.FindByID(
		ctx,
		id,
	)
	if err != nil {
		return err
	}

	if err := s.repository.Delete(
		ctx,
		id,
	); err != nil {
		return err
	}

	if s.cache != nil {
		if err := s.cache.DeleteURL(
			ctx,
			url.ShortCode,
		); err != nil {
			log.Printf(
				"redis cache invalidation failed for %q: %v",
				url.ShortCode,
				err,
			)
		}
	}

	return nil
}
