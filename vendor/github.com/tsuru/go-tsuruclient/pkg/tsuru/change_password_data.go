/*
 * Tsuru
 *
 * Open source, extensible and Docker-based Platform as a Service (PaaS)
 *
 * API version: 1.6
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */

package tsuru

type ChangePasswordData struct {
	Confirm string `json:"confirm,omitempty"`

	New string `json:"new,omitempty"`

	Old string `json:"old,omitempty"`
}
