from setuptools import setup, find_packages

setup(
    name="observation-service",
    version="0.1.0",
    packages=find_packages(),
    install_requires=[
        # List your dependencies here
        "fastapi",
        "uvicorn",
        "python-dotenv",
        "google-cloud-healthcare",
        "pydantic",
    ],
    python_requires=">=3.8",
)
