from setuptools import setup, find_packages

setup(
    name="module8-shared",
    version="1.0.0",
    packages=find_packages(),
    install_requires=[
        "aiokafka>=0.10.0",
        "pydantic>=2.5.0",
        "confluent-kafka>=2.3.0",
    ],
    python_requires=">=3.9",
)
