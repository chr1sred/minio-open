// Copyright (c) 2015-2026 MinIO, Inc.
//
// This file is part of MinIO Object Storage stack
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

// minio-bootstrap is a small, idempotent post-start helper used by the
// container image. It waits for the local MinIO server to be ready and then:
//
//   - creates the buckets listed in MINIO_INIT_BUCKETS
//   - grants anonymous read (download only) on MINIO_INIT_PUBLIC_PATHS
//   - creates an application user (MINIO_INIT_APP_USER) restricted to the
//     buckets above, so apps like Typebot never use the root credentials
//   - prints a step-by-step guide with the values to paste into the app
//
// Running it many times is safe: existing resources are left untouched.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/minio/madmin-go/v3"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

func env(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}

// secretEnv reads KEY, falling back to the file named by KEY_FILE
// (docker secrets style).
func secretEnv(key string) string {
	if v := env(key, ""); v != "" {
		return v
	}
	if f := env(key+"_FILE", ""); f != "" {
		if !strings.HasPrefix(f, "/") {
			f = "/run/secrets/" + f
		}
		if b, err := os.ReadFile(f); err == nil {
			return strings.TrimSpace(string(b))
		}
	}
	return ""
}

func list(v string) (out []string) {
	for _, s := range strings.Split(v, ",") {
		if s = strings.Trim(strings.TrimSpace(s), "/"); s != "" {
			out = append(out, s)
		}
	}
	return out
}

func logf(format string, args ...any) {
	fmt.Printf("[bootstrap] "+format+"\n", args...)
}

func waitReady(ctx context.Context, endpoint string) error {
	u := "http://" + endpoint + "/minio/health/ready"
	for {
		resp, err := http.Get(u)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("MinIO is not ready at %s: %w", u, ctx.Err())
		case <-time.After(2 * time.Second):
		}
	}
}

type statement struct {
	Effect    string         `json:"Effect"`
	Principal map[string]any `json:"Principal,omitempty"`
	Action    []string       `json:"Action"`
	Resource  []string       `json:"Resource"`
}

type policy struct {
	Version   string      `json:"Version"`
	Statement []statement `json:"Statement"`
}

func (p policy) String() string {
	b, _ := json.Marshal(p)
	return string(b)
}

// publicReadPolicy allows anonymous GET (download only, no listing and no
// upload) for the given prefixes of a bucket.
func publicReadPolicy(bucket string, prefixes []string) policy {
	var res []string
	for _, p := range prefixes {
		if p == "" {
			res = append(res, "arn:aws:s3:::"+bucket+"/*")
		} else {
			res = append(res, "arn:aws:s3:::"+bucket+"/"+p+"/*")
		}
	}
	return policy{Version: "2012-10-17", Statement: []statement{
		{Effect: "Allow", Principal: map[string]any{"AWS": []string{"*"}}, Action: []string{"s3:GetBucketLocation"}, Resource: []string{"arn:aws:s3:::" + bucket}},
		{Effect: "Allow", Principal: map[string]any{"AWS": []string{"*"}}, Action: []string{"s3:GetObject"}, Resource: res},
	}}
}

// appPolicy gives full object access, restricted to the given buckets.
func appPolicy(buckets []string) policy {
	var bres, ores []string
	for _, b := range buckets {
		bres = append(bres, "arn:aws:s3:::"+b)
		ores = append(ores, "arn:aws:s3:::"+b+"/*")
	}
	return policy{Version: "2012-10-17", Statement: []statement{
		{Effect: "Allow", Action: []string{"s3:GetBucketLocation", "s3:ListBucket", "s3:ListBucketMultipartUploads"}, Resource: bres},
		{Effect: "Allow", Action: []string{"s3:GetObject", "s3:PutObject", "s3:DeleteObject", "s3:AbortMultipartUpload", "s3:ListMultipartUploadParts"}, Resource: ores},
	}}
}

func run() error {
	if strings.EqualFold(env("MINIO_INIT_ENABLE", "on"), "off") {
		logf("disabled (MINIO_INIT_ENABLE=off)")
		return nil
	}

	endpoint := env("MINIO_INIT_ENDPOINT", "127.0.0.1:9000")
	rootUser, rootPass := secretEnv("MINIO_ROOT_USER"), secretEnv("MINIO_ROOT_PASSWORD")
	if rootUser == "" || rootPass == "" {
		return fmt.Errorf("MINIO_ROOT_USER and MINIO_ROOT_PASSWORD must be set")
	}
	region := env("MINIO_REGION", env("MINIO_SITE_REGION", "us-east-1"))
	buckets := list(env("MINIO_INIT_BUCKETS", "typebot"))
	publicPaths := list(env("MINIO_INIT_PUBLIC_PATHS", "typebot/public"))
	appUser, appPass := env("MINIO_INIT_APP_USER", ""), secretEnv("MINIO_INIT_APP_PASSWORD")
	appBuckets := list(env("MINIO_INIT_APP_BUCKETS", strings.Join(buckets, ",")))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	if err := waitReady(ctx, endpoint); err != nil {
		return err
	}

	s3, err := minio.New(endpoint, &minio.Options{Creds: credentials.NewStaticV4(rootUser, rootPass, ""), Region: region})
	if err != nil {
		return err
	}
	adm, err := madmin.NewWithOptions(endpoint, &madmin.Options{Creds: credentials.NewStaticV4(rootUser, rootPass, "")})
	if err != nil {
		return err
	}

	for _, b := range buckets {
		ok, err := s3.BucketExists(ctx, b)
		if err != nil {
			return fmt.Errorf("bucket %q: %w", b, err)
		}
		if ok {
			logf("bucket %q already exists", b)
			continue
		}
		if err = s3.MakeBucket(ctx, b, minio.MakeBucketOptions{Region: region}); err != nil {
			return fmt.Errorf("creating bucket %q: %w", b, err)
		}
		logf("bucket %q created", b)
	}

	// Group public prefixes per bucket: "typebot/public" -> typebot: [public]
	public := map[string][]string{}
	var order []string
	for _, p := range publicPaths {
		bucket, prefix, _ := strings.Cut(p, "/")
		if _, ok := public[bucket]; !ok {
			order = append(order, bucket)
		}
		public[bucket] = append(public[bucket], prefix)
	}
	for _, bucket := range order {
		want := publicReadPolicy(bucket, public[bucket]).String()
		cur, err := s3.GetBucketPolicy(ctx, bucket)
		if err != nil {
			return fmt.Errorf("reading policy of %q: %w", bucket, err)
		}
		if cur != "" {
			logf("bucket %q already has an access policy, leaving it as is (edit it in the console: Buckets > %s > Anonymous)", bucket, bucket)
			continue
		}
		if err = s3.SetBucketPolicy(ctx, bucket, want); err != nil {
			return fmt.Errorf("setting public read on %q: %w", bucket, err)
		}
		logf("anonymous download enabled on %s/%s", bucket, strings.Join(public[bucket], ", "+bucket+"/"))
	}

	if appUser != "" && appPass != "" {
		policyName := "app-" + appUser + "-rw"
		if err = adm.AddCannedPolicy(ctx, policyName, []byte(appPolicy(appBuckets).String())); err != nil {
			return fmt.Errorf("creating policy %q: %w", policyName, err)
		}
		if _, err = adm.GetUserInfo(ctx, appUser); err == nil {
			logf("user %q already exists (password not changed)", appUser)
		} else {
			if err = adm.AddUser(ctx, appUser, appPass); err != nil {
				return fmt.Errorf("creating user %q: %w", appUser, err)
			}
			logf("user %q created", appUser)
		}
		_, err = adm.AttachPolicy(ctx, madmin.PolicyAssociationReq{Policies: []string{policyName}, User: appUser})
		if err != nil && !strings.Contains(strings.ToLower(err.Error()), "already") {
			return fmt.Errorf("attaching policy to %q: %w", appUser, err)
		}
		logf("policy %q attached to %q (access limited to: %s)", policyName, appUser, strings.Join(appBuckets, ", "))
	}

	printGuide(region, buckets, publicPaths, appUser, appPass != "")
	return nil
}

func printGuide(region string, buckets, publicPaths []string, appUser string, hasPass bool) {
	serverURL := env("MINIO_SERVER_URL", "")
	consoleURL := env("MINIO_BROWSER_REDIRECT_URL", "")

	s3Endpoint, s3SSL, s3Port := "<seu-dominio-s3>", "true", ""
	if u, err := url.Parse(serverURL); err == nil && u.Host != "" {
		s3Endpoint = u.Hostname()
		s3SSL = fmt.Sprint(u.Scheme == "https")
		s3Port = u.Port()
	}
	bucket := "typebot"
	if len(buckets) > 0 {
		bucket = buckets[0]
	}

	line := strings.Repeat("=", 72)
	fmt.Println(line)
	fmt.Println(" MinIO pronto!  /  MinIO is ready!")
	fmt.Println(line)
	if consoleURL != "" {
		fmt.Println(" Console (admin completo) : " + consoleURL)
	}
	if serverURL != "" {
		fmt.Println(" API S3                   : " + serverURL)
	}
	fmt.Println(" Buckets                  : " + strings.Join(buckets, ", "))
	if len(publicPaths) > 0 {
		fmt.Println(" Leitura publica em       : " + strings.Join(publicPaths, ", "))
	}
	fmt.Println(line)
	if appUser == "" || !hasPass {
		fmt.Println(" Nenhum usuario de aplicacao criado. Defina MINIO_INIT_APP_USER e")
		fmt.Println(" MINIO_INIT_APP_PASSWORD e reinicie o app para gera-lo automaticamente.")
		fmt.Println(line)
		return
	}
	fmt.Println(" Cole no Typebot (builder E viewer) -> App Configs > Environment Variables:")
	fmt.Println()
	fmt.Println("   S3_ACCESS_KEY=" + appUser)
	fmt.Println("   S3_SECRET_KEY=<o valor de MINIO_INIT_APP_PASSWORD deste app>")
	fmt.Println("   S3_BUCKET=" + bucket)
	fmt.Println("   S3_ENDPOINT=" + s3Endpoint)
	fmt.Println("   S3_SSL=" + s3SSL)
	if s3Port != "" {
		fmt.Println("   S3_PORT=" + s3Port)
	}
	fmt.Println("   S3_REGION=" + region)
	fmt.Println(line)
	if serverURL == "" {
		fmt.Println(" DICA: defina MINIO_SERVER_URL=https://s3.seudominio.com para links corretos.")
		fmt.Println(line)
	}
}

func main() {
	if err := run(); err != nil {
		logf("ERROR: %v", err)
		os.Exit(1)
	}
}
