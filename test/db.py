import os
import psycopg2

db = psycopg2.connect(
   database=os.environ.get("POSTGRES_DB"),
   user=os.environ.get("POSTGRES_USER"),
   password=os.environ.get("POSTGRES_PASSWORD"),
   host='db',
   port= '5432'
)
