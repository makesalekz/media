## Media:

|     Field      |   Type    | Unique | Optional | Nillable | Default | UpdateDefault | Immutable |            StructTag            | Validators | Comment |
|---|---|---|---|---|---|---|---|---|---|---|
| id             | int       | false  | false    | false    | false   | false         | false     | json:"id,omitempty"             |          0 |         |
| deleted_at     | time.Time | false  | true     | true     | false   | false         | false     | json:"deleted_at,omitempty"     |          0 |         |
| owner_id       | int64     | false  | false    | false    | false   | false         | true      | json:"owner_id,omitempty"       |          0 |         |
| file_name      | string    | false  | false    | false    | false   | false         | true      | json:"file_name,omitempty"      |          0 |         |
| extension      | string    | false  | false    | false    | false   | false         | true      | json:"extension,omitempty"      |          2 |         |
| path           | string    | true   | false    | false    | false   | false         | true      | json:"path,omitempty"           |          0 |         |
| url            | string    | false  | true     | true     | false   | false         | false     | json:"url,omitempty"            |          0 |         |
| size           | int32     | false  | false    | false    | false   | false         | false     | json:"size,omitempty"           |          0 |         |
| width          | int32     | false  | true     | true     | false   | false         | false     | json:"width,omitempty"          |          0 |         |
| height         | int32     | false  | true     | true     | false   | false         | false     | json:"height,omitempty"         |          0 |         |
| duration       | float32   | false  | true     | true     | false   | false         | false     | json:"duration,omitempty"       |          0 |         |
| thumbnail_url  | string    | false  | true     | true     | false   | false         | false     | json:"thumbnail_url,omitempty"  |          0 |         |
| thumbnail_path | string    | false  | true     | true     | false   | false         | false     | json:"thumbnail_path,omitempty" |          0 |         |
| is_activated   | bool      | false  | false    | false    | true    | false         | false     | json:"is_activated,omitempty"   |          0 |         |
| created_at     | time.Time | false  | false    | false    | true    | false         | true      | json:"created_at,omitempty"     |          0 |         |
| uploaded_at    | time.Time | false  | true     | true     | false   | false         | false     | json:"uploaded_at,omitempty"    |          0 |         |

