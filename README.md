# Gitlab Repo Extractor

This project is a go-based tool for cloning a GitLab repositories from specified groups.  
It fetches groups and projects using the GitLab API, clones them to a local directory.

## Installation
1. Clone this repository:
   ```sh
   git clone https://github.com/artzub/gitlab-repo-extractor.git
   cd gitlab-repo-extractor
   ```
2. Build the project with using `make`:
    ```sh
    make build
    ```
3. Build the project manually:
   1. Install modules.
      ```sh
      go mod tidy
      ```
   2. Build the binary.
      ```sh
      go build .
      ```

## Usage
Create a `.env.local` file in the root directory with the following content:
```env
RE_GITLAB_TOKEN=<your_gitlab_token>
```
List of available environment variables:

| Variable                   | Description                                                                                 | Default            | Example                                     |
|----------------------------|---------------------------------------------------------------------------------------------|--------------------|---------------------------------------------|
| **RE_GITLAB_URL**          | GitLab Server URL                                                                           | https://gitlab.com | `RE_GITLAB_URL=https://gitlab.com`          |
| **RE_GITLAB_TOKEN**        | GitLab API access token                                                                     |                    | `RE_GITLAB_TOKEN=xxxx`                      |
| **RE_OUTPUT_DIR**          | Output directory                                                                            | ./gitlab-repos     | `RE_OUTPUT_DIR=./gitlab-repos`              |
| **RE_MAX_WORKERS**         | Number of workers                                                                           | `runtime.NumCPU()` | `RE_MAX_WORKERS=5`                          |
| **RE_MAX_RETRIES**         | Retry attempts for failed clones                                                            | 3                  | `RE_MAX_RETRIES=3`                          |
| **RE_RETRY_DELAY_SECONDS** | Delay between retries in seconds                                                            | 2                  | `RE_RETRY_DELAY_SECONDS=2`                  |
| **RE_GROUP_IDS**           | Clone specific groups only, split by comma or space.<br/>[More about group ids](#group-ids) |                    | `RE_GROUP_IDS="gitlab-org, gitlab-org/api"` |
| **RE_SKIP_GROUP_IDS**      | Skip specific groups, split by comma or space<br/>[More about group ids](#group-ids)        |                    | `RE_SKIP_GROUP_IDS="gitlab-org/api"`        |
| **RE_USE_SSH**             | Use SSH for cloning                                                                         | false              | `RE_USE_SSH=false`                          |
| **RE_CLONE_BARE**          | Use bare cloning (no working directory)                                                     | true               | `RE_CLONE_BARE=true`                        |

### Group IDs
Group ID can be the integer ID of group or a path to the group [URL-encoded path of the group](https://docs.gitlab.com/api/rest/#namespaced-paths).    
`<gitlab-server-url>/<group-path>/<project-path>`  
`<group-path>` - `<group-name>/<sub-group-name>`  
For example: `gitlab-org/api` - group with path `https://gitlab.org/gitlab-org/api`


## Development
- Ensure you have Go installed (version 1.24 or later).
- Before commiting
    ```sh
    make reviewable
    ```

## License
MIT

## Contributing
Pull requests are welcome! Please add tests for new features and ensure all tests pass.

