Initial Setup

Orchestrator:
    - Fills the MongoDB
    - Setups the Workers in list based on their IP:Port
    - Based on the API receives the task for CosSim for a User
    - Setups the task and sends to the workers based on their availability
    - If a worker doesn't respond based on their ping after x time it restarts the job to another worker and deletes that worker. If the workers don't exist sends a error to the API.

Worker:
    - Receives the task from the orchestrator
    - Does the Job
    - Receives the data from the MongoDB
    - Send their data to Redis
    - Send the orchestator if it is done or not

JSON Schemas
Worker Job
Job {
    "Job": "(Type of job):integer",
    "Inf": "Information of the job",
}

API
/login
/register

/me
/userPanel
/metricsPanel

/topMovies
/userSimilarity
