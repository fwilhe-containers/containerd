/*
   Copyright The containerd Authors.

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

package encryption

import (
	"context"
	"fmt"

	containerd "github.com/containerd/containerd/v2/client"
	"github.com/containerd/containerd/v2/core/containers"
	"github.com/containerd/containerd/v2/core/diff"
	"github.com/containerd/errdefs"
	"github.com/containerd/typeurl/v2"

	"github.com/containerd/imgcrypt/v2"

	encconfig "github.com/containers/ocicrypt/config"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

// WithDecryptedUnpack allows to pass parameters the 'layertool' needs to the applier
func WithDecryptedUnpack(data *imgcrypt.Payload) diff.ApplyOpt {
	return func(_ context.Context, desc ocispec.Descriptor, c *diff.ApplyConfig) error {
		data.Descriptor = desc
		anything, err := typeurl.MarshalAny(data)
		if err != nil {
			return fmt.Errorf("failed to marshal payload: %w", err)
		}

		if c.ProcessorPayloads == nil {
			c.ProcessorPayloads = make(map[string]typeurl.Any, len(imgcrypt.PayloadToolIDs))
		}
		for _, id := range imgcrypt.PayloadToolIDs {
			c.ProcessorPayloads[id] = anything
		}
		return nil
	}
}

// WithUnpackConfigApplyOpts allows to pass an ApplyOpt
func WithUnpackConfigApplyOpts(opt diff.ApplyOpt) containerd.UnpackOpt {
	return func(_ context.Context, uc *containerd.UnpackConfig) error {
		uc.ApplyOpts = append(uc.ApplyOpts, opt)
		return nil
	}
}

// WithUnpackOpts is used to add unpack options to the unpacker.
func WithUnpackOpts(opts []containerd.UnpackOpt) containerd.RemoteOpt {
	return func(_ *containerd.Client, c *containerd.RemoteContext) error {
		c.UnpackOpts = append(c.UnpackOpts, opts...)
		return nil
	}
}

// WithAuthorizationCheck checks the authorization of keys used for encrypted containers
// be checked upon creation of a container
func WithAuthorizationCheck(dc *encconfig.DecryptConfig) containerd.NewContainerOpts {
	return func(ctx context.Context, client *containerd.Client, c *containers.Container) error {
		image, err := client.ImageService().Get(ctx, c.Image)
		if errdefs.IsNotFound(err) {
			// allow creation of container without a existing image
			return nil
		} else if err != nil {
			return err
		}

		return CheckAuthorization(ctx, client.ContentStore(), image.Target, dc)
	}
}
