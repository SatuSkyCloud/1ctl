# Microservices example

This example contains two independently deployable applications:

- `catalog` serves `GET /item` on port 8001.
- `checkout` serves `GET /order-preview` on port 8000 and calls the catalog URL
  supplied in `CATALOG_URL`.

Deploy each application from its own directory. See the
[microservices guide](https://docs.satusky.com/guides/microservices/) for the
complete deployment, verification, failure-isolation, and cleanup flow.
