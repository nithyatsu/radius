/*
Copyright 2023 The Radius Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package database

type (
	// QueryOptions applies an option to Query().
	QueryOptions interface {
		ApplyQueryOption(DatabaseOptions) DatabaseOptions

		// A private method to prevent users implementing the
		// interface and so future additions to it will not
		// violate compatibility.
		private()
	}

	// GetOptions applies an option to Get().
	GetOptions interface {
		// A private method to prevent users implementing the
		// interface and so future additions to it will not
		// violate compatibility.
		private()
	}

	// DeleteOptions applies an option to Delete().
	DeleteOptions interface {
		ApplyDeleteOption(DatabaseOptions) DatabaseOptions

		// A private method to prevent users implementing the
		// interface and so future additions to it will not
		// violate compatibility.
		private()
	}

	// SaveOptions applies an option to Save().
	SaveOptions interface {
		ApplySaveOption(DatabaseOptions) DatabaseOptions

		// A private method to prevent users implementing the
		// interface and so future additions to it will not
		// violate compatibility.
		private()
	}

	// MutatingOptions applies an option to Delete() or Save().
	MutatingOptions interface {
		SaveOptions
		DeleteOptions
	}
)

// DatabaseOptions represents the configurations of the underlying database APIs.
type DatabaseOptions struct {
	// PaginationToken represents pagination token such as continuation token.
	PaginationToken string

	// MaxQueryItemCount represents max items in query result.
	MaxQueryItemCount int

	// ETag represents the entity tag for optimistic consistency control.
	ETag ETag
}

// Query Options
type queryOptions struct {
	fn func(DatabaseOptions) DatabaseOptions
}

// ApplyQueryOption applies a function to the StoreConfig to modify it.
func (q *queryOptions) ApplyQueryOption(cfg DatabaseOptions) DatabaseOptions {
	return q.fn(cfg)
}

func (q queryOptions) private() {}

// WithPaginationToken sets pagination token for Query().
func WithPaginationToken(token string) QueryOptions {
	return &queryOptions{
		fn: func(cfg DatabaseOptions) DatabaseOptions {
			cfg.PaginationToken = token
			return cfg
		},
	}
}

// WithMaxQueryItemCount creates a QueryOptions instance that sets the maximum number of items in query result.
func WithMaxQueryItemCount(maxcnt int) QueryOptions {
	return &queryOptions{
		fn: func(cfg DatabaseOptions) DatabaseOptions {
			cfg.MaxQueryItemCount = maxcnt
			return cfg
		},
	}
}

// MutatingOptions
type mutatingOptions struct {
	fn func(DatabaseOptions) DatabaseOptions
}

var _ DeleteOptions = (*mutatingOptions)(nil)
var _ SaveOptions = (*mutatingOptions)(nil)

// ApplyDeleteOption applies the delete option to the given StoreConfig and returns the modified StoreConfig.
func (s *mutatingOptions) ApplyDeleteOption(cfg DatabaseOptions) DatabaseOptions {
	return s.fn(cfg)
}

// ApplySaveOption applies the save option to the given StoreConfig and returns the modified StoreConfig.
func (s *mutatingOptions) ApplySaveOption(cfg DatabaseOptions) DatabaseOptions {
	return s.fn(cfg)
}

func (s mutatingOptions) private() {}

// SaveOptions
type saveOptions struct {
	fn func(DatabaseOptions) DatabaseOptions
}

var _ SaveOptions = (*saveOptions)(nil)

// ApplySaveOption applies a save option to a StoreConfig.
func (s *saveOptions) ApplySaveOption(cfg DatabaseOptions) DatabaseOptions {
	return s.fn(cfg)
}

func (s saveOptions) private() {}

// WithETag sets the ETag field in the StoreConfig struct.
func WithETag(etag ETag) MutatingOptions {
	return &mutatingOptions{
		fn: func(cfg DatabaseOptions) DatabaseOptions {
			cfg.ETag = etag
			return cfg
		},
	}
}

// NewQueryConfig applies a set of QueryOptions to a StoreConfig and returns the modified StoreConfig for Query().
func NewQueryConfig(opts ...QueryOptions) DatabaseOptions {
	cfg := DatabaseOptions{}
	for _, opt := range opts {
		cfg = opt.ApplyQueryOption(cfg)
	}
	return cfg
}

// NewDeleteConfig applies the given DeleteOptions to a StoreConfig and returns the resulting StoreConfig for Delete().
func NewDeleteConfig(opts ...DeleteOptions) DatabaseOptions {
	cfg := DatabaseOptions{}
	for _, opt := range opts {
		cfg = opt.ApplyDeleteOption(cfg)
	}
	return cfg
}

// NewSaveConfig applies a set of SaveOptions to a StoreConfig and returns the modified StoreConfig for Save().
func NewSaveConfig(opts ...SaveOptions) DatabaseOptions {
	cfg := DatabaseOptions{}
	for _, opt := range opts {
		cfg = opt.ApplySaveOption(cfg)
	}
	return cfg
}
