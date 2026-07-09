/*
Copyright 2026 pixiv Inc.

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

package controller

import (
	"context"
	"reflect"

	pixivnetv1 "github.com/pixiv/k8s-job-wrapper/api/v1"
	"github.com/pixiv/k8s-job-wrapper/internal/construct"
	batchv1 "k8s.io/api/batch/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func ensureInstance[T any]() T {
	var zero T
	if t := reflect.TypeOf((*T)(nil)).Elem(); t.Kind() == reflect.Ptr {
		return reflect.New(t.Elem()).Interface().(T)
	}
	return zero
}

func Get[T client.Object](ctx context.Context, c client.Client, key client.ObjectKey, opts ...client.GetOption) (T, error) {
	x := ensureInstance[T]()
	if err := c.Get(ctx, key, x, opts...); err != nil {
		return x, err
	}
	return x, nil
}

func List[T client.ObjectList](ctx context.Context, c client.Client, opts ...client.ListOption) (T, error) {
	x := ensureInstance[T]()
	if err := c.List(ctx, x, opts...); err != nil {
		return x, err
	}
	return x, nil
}

func Exist[T client.Object](ctx context.Context, c client.Client, key client.ObjectKey, opts ...client.GetOption) (bool, error) {
	_, err := Get[T](ctx, c, key, opts...)
	switch {
	case err == nil:
		return true, nil
	case apierrors.IsNotFound(err):
		return false, nil
	default:
		return false, err
	}
}

func GetBatchCronJobFromPixivNetCronJob(ctx context.Context, c client.Client, cronJob *pixivnetv1.CronJob) (*batchv1.CronJob, error) {
	return Get[*batchv1.CronJob](ctx, c, types.NamespacedName{
		Namespace: cronJob.Namespace,
		Name:      construct.BatchCronJobName(cronJob),
	})
}
