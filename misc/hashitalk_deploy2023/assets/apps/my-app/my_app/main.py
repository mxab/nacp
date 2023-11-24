from fastapi import FastAPI
import psycopg2 as pg  # type: ignore
import psycopg2.extras  # type: ignore
import logging

logger = logging.getLogger("my-app")

app = FastAPI()


@app.get("/")
def hello(name: str = "World"):
    logger.info("Greeting %s", name)

    message = f"Hello {name}!"

    sql("INSERT INTO greetings (message) VALUES (%(message)s)", {"message": message})

    return message


@app.get("/greetings")
def greetings():
    logger.info("Fetching all greetings")
    return sql("SELECT * FROM greetings", fetch=True)


def create_table():
    logger.info("Creating table if not exist...")
    sql(
        """
        CREATE TABLE IF NOT EXISTS greetings
        (id SERIAL PRIMARY KEY, message VARCHAR(255) NOT NULL);"""
    )


def sql(query: str, vars=None, fetch: bool = False):

    try:
        connection = pg.connect("postgresql://")  # pglib env vars
        # this is not prod code ;)
        connection.set_session(autocommit=True)
        with connection.cursor(cursor_factory=pg.extras.RealDictCursor) as c:
            c.execute(query=query, vars=vars)
            if fetch:
                return c.fetchall()
        connection.close()
    except Exception as e:
        logger.warning("Failed to execute query: %s, error: %s", query, str(e))


if __name__ == "__main__":
    import uvicorn

    logging.basicConfig(level=logging.INFO)
    create_table()
    logger.info("Starting my app...")
    uvicorn.run(app, host="0.0.0.0", port=8000)
