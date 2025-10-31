### Building and running your application

When you're ready, start your application by running:
`docker compose up --build`.

### Data Persistence

The SQLite database is persisted using Docker volumes. The database file is stored in `./app/data/db.sqlite` on your host machine and is mounted to `/app/data` in the container.

**Important notes:**
- The database will persist across container restarts and rebuilds
- To reset the database, simply delete `./app/data/db.sqlite` and restart the container
- The `app/data` directory must exist on your host (a `.gitkeep` file maintains it in git)
- Database files are ignored by git (see `.gitignore`)


### Deploying your application to the cloud

First, build your image, e.g.: `docker build -t myapp .`.
If your cloud uses a different CPU architecture than your development
machine (e.g., you are on a Mac M1 and your cloud provider is amd64),
you'll want to build the image for that platform, e.g.:
`docker build --platform=linux/amd64 -t myapp .`.

Then, push it to your registry, e.g. `docker push myregistry.com/myapp`.

Consult Docker's [getting started](https://docs.docker.com/go/get-started-sharing/)
docs for more detail on building and pushing.

### References
* [Docker's Go guide](https://docs.docker.com/language/golang/)