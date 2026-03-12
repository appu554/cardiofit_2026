from setuptools import setup, find_packages

setup(
    name="shared",
    version="0.1.0",
    packages=find_packages(),
    description="Shared utilities for Clinical Synthesis Hub microservices",
    author="Clinical Synthesis Hub Team",
    author_email="admin@example.com",
    install_requires=[
        "fastapi>=0.95.0",
        "starlette>=0.27.0",
        "python-jose>=3.3.0",
        "requests>=2.28.0",
    ],
)
