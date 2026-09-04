from fastapi import FastAPI

app = FastAPI()


@app.get("/health")
def health() -> dict[str, str]:
    return {"status": "ok", "service": "python-fastapi"}


@app.get("/")
def root() -> dict[str, str]:
    return {"message": "Hello from FastAPI"}
