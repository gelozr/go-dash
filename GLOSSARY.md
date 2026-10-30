# go-dash

An invoicing dashboard API: staff log in to track Customers, their Invoices, and revenue.

## Language

### People

**User**:
A staff member who logs in to the dashboard.
_Avoid_: Customer, account

**Customer**:
A person or business that Invoices are issued to. Customers never log in.
_Avoid_: Client, User

### Authentication

**Access Token**:
A short-lived credential that proves which User is making a request.
_Avoid_: JWT, bearer, session

**Refresh Session**:
A long-lived, single-use grant that lets a User get a new Access Token without logging in again. Using it replaces it with a new one.
_Avoid_: Refresh token (that is only the Refresh Session's identifier), session

**Token Guard**:
The authentication module a User goes through to log in, refresh, and have their Access Token verified.
_Avoid_: Auth manager, auth service, guard name

**Token Provider**:
The part of the Token Guard that decides how Access Tokens and Refresh Sessions are issued, refreshed and verified.
_Avoid_: Driver, token service
