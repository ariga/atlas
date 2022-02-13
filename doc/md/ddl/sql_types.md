---
id: ddl-sql-types
title: SQL Column Types
slug: /ddl/sql-types
---

### MySQL/MariaDB

<table>
    <thead>
    <tr>
        <th>HCL</th>
        <th>SQL</th>
        <th>Attributes</th>
        <th>Example</th>
    </tr>
    </thead>
    <tbody>
        
        <tr>
            <td>bigint</td>
            <td>bigint</td>
            <td>
                <ul>
                        <li>unsigned (bool)</li>
                        <li>size (int)</li>
                </ul>
            </td>
            <td>
                
                <pre>
                    type = bigint(255)
                </pre>
                
                <pre>
                    unsigned = true
                </pre>
                
                
            </td>
        </tr>
        
        <tr>
            <td>binary</td>
            <td>binary</td>
            <td>
                <ul>
                        <li>size (int)</li>
                </ul>
            </td>
            <td>
                
                <pre>
                    type = binary(255)
                </pre>
                
                
            </td>
        </tr>
        
        <tr>
            <td>bit</td>
            <td>bit</td>
            <td>
                <ul>
                        <li>size (int)</li>
                </ul>
            </td>
            <td>
                
                <pre>
                    type = bit(255)
                </pre>
                
                
            </td>
        </tr>
        
        <tr>
            <td>blob</td>
            <td>blob</td>
            <td>
                <ul>
                        <li>size (int)</li>
                </ul>
            </td>
            <td>
                
                <pre>
                    type = blob(255)
                </pre>
                
                
            </td>
        </tr>
        
        <tr>
            <td>bool</td>
            <td>bool</td>
            <td>
                <ul>
                </ul>
            </td>
            <td>
                
                <pre>
                    type = bool
                </pre>
                
                
            </td>
        </tr>
        
        <tr>
            <td>boolean</td>
            <td>boolean</td>
            <td>
                <ul>
                </ul>
            </td>
            <td>
                
                <pre>
                    type = boolean
                </pre>
                
                
            </td>
        </tr>
        
        <tr>
            <td>char</td>
            <td>char</td>
            <td>
                <ul>
                        <li>size (int)</li>
                </ul>
            </td>
            <td>
                
                <pre>
                    type = char(255)
                </pre>
                
                
            </td>
        </tr>
        
        <tr>
            <td>date</td>
            <td>date</td>
            <td>
                <ul>
                        <li>precision (int)</li>
                </ul>
            </td>
            <td>
                
                <pre>
                    type = date(10)
                </pre>
                
                
            </td>
        </tr>
        
        <tr>
            <td>datetime</td>
            <td>datetime</td>
            <td>
                <ul>
                        <li>precision (int)</li>
                </ul>
            </td>
            <td>
                
                <pre>
                    type = datetime(10)
                </pre>
                
                
            </td>
        </tr>
        
        <tr>
            <td>decimal</td>
            <td>decimal</td>
            <td>
                <ul>
                        <li>unsigned (bool)</li>
                        <li>precision (int)</li>
                        <li>scale (int)</li>
                </ul>
            </td>
            <td>
                
                <pre>
                    type = decimal(10,2)
                </pre>
                
                <pre>
                    unsigned = true
                </pre>
                
                
            </td>
        </tr>
        
        <tr>
            <td>double</td>
            <td>double</td>
            <td>
                <ul>
                        <li>unsigned (bool)</li>
                        <li>precision (int)</li>
                        <li>scale (int)</li>
                </ul>
            </td>
            <td>
                
                <pre>
                    type = double(10,2)
                </pre>
                
                <pre>
                    unsigned = true
                </pre>
                
                
            </td>
        </tr>
        
        <tr>
            <td>enum</td>
            <td>enum</td>
            <td>
                <ul>
                        <li>values (slice)</li>
                </ul>
            </td>
            <td>
                
                <pre>
                    type = enum("a","b")
                </pre>
                
                
            </td>
        </tr>
        
        <tr>
            <td>float</td>
            <td>float</td>
            <td>
                <ul>
                        <li>unsigned (bool)</li>
                        <li>precision (int)</li>
                        <li>scale (int)</li>
                </ul>
            </td>
            <td>
                
                <pre>
                    type = float(10,2)
                </pre>
                
                <pre>
                    unsigned = true
                </pre>
                
                
            </td>
        </tr>
        
        <tr>
            <td>geometry</td>
            <td>geometry</td>
            <td>
                <ul>
                </ul>
            </td>
            <td>
                
                <pre>
                    type = geometry
                </pre>
                
                
            </td>
        </tr>
        
        <tr>
            <td>geometrycollection</td>
            <td>geometrycollection</td>
            <td>
                <ul>
                </ul>
            </td>
            <td>
                
                <pre>
                    type = geometrycollection
                </pre>
                
                
            </td>
        </tr>
        
        <tr>
            <td>int</td>
            <td>int</td>
            <td>
                <ul>
                        <li>unsigned (bool)</li>
                        <li>size (int)</li>
                </ul>
            </td>
            <td>
                
                <pre>
                    type = int(255)
                </pre>
                
                <pre>
                    unsigned = true
                </pre>
                
                
            </td>
        </tr>
        
        <tr>
            <td>json</td>
            <td>json</td>
            <td>
                <ul>
                </ul>
            </td>
            <td>
                
                <pre>
                    type = json
                </pre>
                
                
            </td>
        </tr>
        
        <tr>
            <td>linestring</td>
            <td>linestring</td>
            <td>
                <ul>
                </ul>
            </td>
            <td>
                
                <pre>
                    type = linestring
                </pre>
                
                
            </td>
        </tr>
        
        <tr>
            <td>longblob</td>
            <td>longblob</td>
            <td>
                <ul>
                </ul>
            </td>
            <td>
                
                <pre>
                    type = longblob
                </pre>
                
                
            </td>
        </tr>
        
        <tr>
            <td>longtext</td>
            <td>longtext</td>
            <td>
                <ul>
                </ul>
            </td>
            <td>
                
                <pre>
                    type = longtext
                </pre>
                
                
            </td>
        </tr>
        
        <tr>
            <td>mediumblob</td>
            <td>mediumblob</td>
            <td>
                <ul>
                </ul>
            </td>
            <td>
                
                <pre>
                    type = mediumblob
                </pre>
                
                
            </td>
        </tr>
        
        <tr>
            <td>mediumint</td>
            <td>mediumint</td>
            <td>
                <ul>
                        <li>unsigned (bool)</li>
                        <li>size (int)</li>
                </ul>
            </td>
            <td>
                
                <pre>
                    type = mediumint(255)
                </pre>
                
                <pre>
                    unsigned = true
                </pre>
                
                
            </td>
        </tr>
        
        <tr>
            <td>mediumtext</td>
            <td>mediumtext</td>
            <td>
                <ul>
                </ul>
            </td>
            <td>
                
                <pre>
                    type = mediumtext
                </pre>
                
                
            </td>
        </tr>
        
        <tr>
            <td>multilinestring</td>
            <td>multilinestring</td>
            <td>
                <ul>
                </ul>
            </td>
            <td>
                
                <pre>
                    type = multilinestring
                </pre>
                
                
            </td>
        </tr>
        
        <tr>
            <td>multipoint</td>
            <td>multipoint</td>
            <td>
                <ul>
                </ul>
            </td>
            <td>
                
                <pre>
                    type = multipoint
                </pre>
                
                
            </td>
        </tr>
        
        <tr>
            <td>multipolygon</td>
            <td>multipolygon</td>
            <td>
                <ul>
                </ul>
            </td>
            <td>
                
                <pre>
                    type = multipolygon
                </pre>
                
                
            </td>
        </tr>
        
        <tr>
            <td>numeric</td>
            <td>numeric</td>
            <td>
                <ul>
                        <li>unsigned (bool)</li>
                        <li>precision (int)</li>
                        <li>scale (int)</li>
                </ul>
            </td>
            <td>
                
                <pre>
                    type = numeric(10,2)
                </pre>
                
                <pre>
                    unsigned = true
                </pre>
                
                
            </td>
        </tr>
        
        <tr>
            <td>point</td>
            <td>point</td>
            <td>
                <ul>
                </ul>
            </td>
            <td>
                
                <pre>
                    type = point
                </pre>
                
                
            </td>
        </tr>
        
        <tr>
            <td>polygon</td>
            <td>polygon</td>
            <td>
                <ul>
                </ul>
            </td>
            <td>
                
                <pre>
                    type = polygon
                </pre>
                
                
            </td>
        </tr>
        
        <tr>
            <td>real</td>
            <td>real</td>
            <td>
                <ul>
                        <li>unsigned (bool)</li>
                        <li>precision (int)</li>
                        <li>scale (int)</li>
                </ul>
            </td>
            <td>
                
                <pre>
                    type = real(10,2)
                </pre>
                
                <pre>
                    unsigned = true
                </pre>
                
                
            </td>
        </tr>
        
        <tr>
            <td>set</td>
            <td>set</td>
            <td>
                <ul>
                        <li>values (slice)</li>
                </ul>
            </td>
            <td>
                
                <pre>
                    type = set("a","b")
                </pre>
                
                
            </td>
        </tr>
        
        <tr>
            <td>smallint</td>
            <td>smallint</td>
            <td>
                <ul>
                        <li>unsigned (bool)</li>
                        <li>size (int)</li>
                </ul>
            </td>
            <td>
                
                <pre>
                    type = smallint(255)
                </pre>
                
                <pre>
                    unsigned = true
                </pre>
                
                
            </td>
        </tr>
        
        <tr>
            <td>text</td>
            <td>text</td>
            <td>
                <ul>
                        <li>size (int)</li>
                </ul>
            </td>
            <td>
                
                <pre>
                    type = text(255)
                </pre>
                
                
            </td>
        </tr>
        
        <tr>
            <td>time</td>
            <td>time</td>
            <td>
                <ul>
                        <li>precision (int)</li>
                </ul>
            </td>
            <td>
                
                <pre>
                    type = time(10)
                </pre>
                
                
            </td>
        </tr>
        
        <tr>
            <td>timestamp</td>
            <td>timestamp</td>
            <td>
                <ul>
                        <li>precision (int)</li>
                </ul>
            </td>
            <td>
                
                <pre>
                    type = timestamp(10)
                </pre>
                
                
            </td>
        </tr>
        
        <tr>
            <td>tinyblob</td>
            <td>tinyblob</td>
            <td>
                <ul>
                </ul>
            </td>
            <td>
                
                <pre>
                    type = tinyblob
                </pre>
                
                
            </td>
        </tr>
        
        <tr>
            <td>tinyint</td>
            <td>tinyint</td>
            <td>
                <ul>
                        <li>unsigned (bool)</li>
                        <li>size (int)</li>
                </ul>
            </td>
            <td>
                
                <pre>
                    type = tinyint(255)
                </pre>
                
                <pre>
                    unsigned = true
                </pre>
                
                
            </td>
        </tr>
        
        <tr>
            <td>tinytext</td>
            <td>tinytext</td>
            <td>
                <ul>
                </ul>
            </td>
            <td>
                
                <pre>
                    type = tinytext
                </pre>
                
                
            </td>
        </tr>
        
        <tr>
            <td>varbinary</td>
            <td>varbinary</td>
            <td>
                <ul>
                        <li>size (int)</li>
                </ul>
            </td>
            <td>
                
                <pre>
                    type = varbinary(255)
                </pre>
                
                
            </td>
        </tr>
        
        <tr>
            <td>varchar</td>
            <td>varchar</td>
            <td>
                <ul>
                        <li>size (int)</li>
                </ul>
            </td>
            <td>
                
                <pre>
                    type = varchar(255)
                </pre>
                
                
            </td>
        </tr>
        
        <tr>
            <td>year</td>
            <td>year</td>
            <td>
                <ul>
                        <li>precision (int)</li>
                </ul>
            </td>
            <td>
                
                <pre>
                    type = year(10)
                </pre>
                
                
            </td>
        </tr>
        
    </tbody>
</table>

### Postgres

<table>
    <thead>
    <tr>
        <th>HCL</th>
        <th>SQL</th>
        <th>Attributes</th>
        <th>Example</th>
    </tr>
    </thead>
    <tbody>
        
        <tr>
            <td>bigint</td>
            <td>bigint</td>
            <td>
                <ul>
                </ul>
            </td>
            <td>
                
                <pre>
                    type = bigint
                </pre>
                
                
            </td>
        </tr>
        
        <tr>
            <td>bigserial</td>
            <td>bigserial</td>
            <td>
                <ul>
                </ul>
            </td>
            <td>
                
                <pre>
                    type = bigserial
                </pre>
                
                
            </td>
        </tr>
        
        <tr>
            <td>bit</td>
            <td>bit</td>
            <td>
                <ul>
                        <li>len (int64)</li>
                </ul>
            </td>
            <td>
                
                <pre>
                    type = bit(255)
                </pre>
                
                
            </td>
        </tr>
        
        <tr>
            <td>bit_varying</td>
            <td>bit varying</td>
            <td>
                <ul>
                        <li>len (int64)</li>
                </ul>
            </td>
            <td>
                
                <pre>
                    type = bit_varying(255)
                </pre>
                
                
            </td>
        </tr>
        
        <tr>
            <td>bool</td>
            <td>bool</td>
            <td>
                <ul>
                </ul>
            </td>
            <td>
                
                <pre>
                    type = bool
                </pre>
                
                
            </td>
        </tr>
        
        <tr>
            <td>boolean</td>
            <td>boolean</td>
            <td>
                <ul>
                </ul>
            </td>
            <td>
                
                <pre>
                    type = boolean
                </pre>
                
                
            </td>
        </tr>
        
        <tr>
            <td>box</td>
            <td>box</td>
            <td>
                <ul>
                </ul>
            </td>
            <td>
                
                <pre>
                    type = box
                </pre>
                
                
            </td>
        </tr>
        
        <tr>
            <td>bytea</td>
            <td>bytea</td>
            <td>
                <ul>
                </ul>
            </td>
            <td>
                
                <pre>
                    type = bytea
                </pre>
                
                
            </td>
        </tr>
        
        <tr>
            <td>char</td>
            <td>char</td>
            <td>
                <ul>
                        <li>size (int)</li>
                </ul>
            </td>
            <td>
                
                <pre>
                    type = char(255)
                </pre>
                
                
            </td>
        </tr>
        
        <tr>
            <td>character</td>
            <td>character</td>
            <td>
                <ul>
                        <li>size (int)</li>
                </ul>
            </td>
            <td>
                
                <pre>
                    type = character(255)
                </pre>
                
                
            </td>
        </tr>
        
        <tr>
            <td>character_varying</td>
            <td>character varying</td>
            <td>
                <ul>
                        <li>size (int)</li>
                </ul>
            </td>
            <td>
                
                <pre>
                    type = character_varying(255)
                </pre>
                
                
            </td>
        </tr>
        
        <tr>
            <td>cidr</td>
            <td>cidr</td>
            <td>
                <ul>
                </ul>
            </td>
            <td>
                
                <pre>
                    type = cidr
                </pre>
                
                
            </td>
        </tr>
        
        <tr>
            <td>circle</td>
            <td>circle</td>
            <td>
                <ul>
                </ul>
            </td>
            <td>
                
                <pre>
                    type = circle
                </pre>
                
                
            </td>
        </tr>
        
        <tr>
            <td>date</td>
            <td>date</td>
            <td>
                <ul>
                </ul>
            </td>
            <td>
                
                <pre>
                    type = date
                </pre>
                
                
            </td>
        </tr>
        
        <tr>
            <td>decimal</td>
            <td>decimal</td>
            <td>
                <ul>
                </ul>
            </td>
            <td>
                
                <pre>
                    type = decimal
                </pre>
                
                
            </td>
        </tr>
        
        <tr>
            <td>double_precision</td>
            <td>double precision</td>
            <td>
                <ul>
                </ul>
            </td>
            <td>
                
                <pre>
                    type = double_precision
                </pre>
                
                
            </td>
        </tr>
        
        <tr>
            <td>enum</td>
            <td>my_enum</td>
            <td>
                <ul>
                </ul>
            </td>
            <td>
                
                <pre>
                    enum = enum.my_enum
                </pre>
                
                In Postgres an enum type is created as a custom type and can then be referenced in a column 
definition. Therefore, you have to add an enum block to your HCL schema like below:
<pre>
enum "my_enum" &#123;
	values = ["on", "off"]
&#125;
</pre>
            </td>
        </tr>
        
        <tr>
            <td>float4</td>
            <td>float4</td>
            <td>
                <ul>
                </ul>
            </td>
            <td>
                
                <pre>
                    type = float4
                </pre>
                
                
            </td>
        </tr>
        
        <tr>
            <td>float8</td>
            <td>float8</td>
            <td>
                <ul>
                </ul>
            </td>
            <td>
                
                <pre>
                    type = float8
                </pre>
                
                
            </td>
        </tr>
        
        <tr>
            <td>hstore</td>
            <td>hstore</td>
            <td>
                <ul>
                </ul>
            </td>
            <td>
                
                <pre>
                    type = hstore
                </pre>
                
                
            </td>
        </tr>
        
        <tr>
            <td>inet</td>
            <td>inet</td>
            <td>
                <ul>
                </ul>
            </td>
            <td>
                
                <pre>
                    type = inet
                </pre>
                
                
            </td>
        </tr>
        
        <tr>
            <td>int</td>
            <td>int</td>
            <td>
                <ul>
                </ul>
            </td>
            <td>
                
                <pre>
                    type = int
                </pre>
                
                
            </td>
        </tr>
        
        <tr>
            <td>int2</td>
            <td>int2</td>
            <td>
                <ul>
                </ul>
            </td>
            <td>
                
                <pre>
                    type = int2
                </pre>
                
                
            </td>
        </tr>
        
        <tr>
            <td>int4</td>
            <td>int4</td>
            <td>
                <ul>
                </ul>
            </td>
            <td>
                
                <pre>
                    type = int4
                </pre>
                
                
            </td>
        </tr>
        
        <tr>
            <td>int8</td>
            <td>int8</td>
            <td>
                <ul>
                </ul>
            </td>
            <td>
                
                <pre>
                    type = int8
                </pre>
                
                
            </td>
        </tr>
        
        <tr>
            <td>integer</td>
            <td>integer</td>
            <td>
                <ul>
                </ul>
            </td>
            <td>
                
                <pre>
                    type = integer
                </pre>
                
                
            </td>
        </tr>
        
        <tr>
            <td>json</td>
            <td>json</td>
            <td>
                <ul>
                </ul>
            </td>
            <td>
                
                <pre>
                    type = json
                </pre>
                
                
            </td>
        </tr>
        
        <tr>
            <td>jsonb</td>
            <td>jsonb</td>
            <td>
                <ul>
                </ul>
            </td>
            <td>
                
                <pre>
                    type = jsonb
                </pre>
                
                
            </td>
        </tr>
        
        <tr>
            <td>line</td>
            <td>line</td>
            <td>
                <ul>
                </ul>
            </td>
            <td>
                
                <pre>
                    type = line
                </pre>
                
                
            </td>
        </tr>
        
        <tr>
            <td>lseg</td>
            <td>lseg</td>
            <td>
                <ul>
                </ul>
            </td>
            <td>
                
                <pre>
                    type = lseg
                </pre>
                
                
            </td>
        </tr>
        
        <tr>
            <td>macaddr</td>
            <td>macaddr</td>
            <td>
                <ul>
                </ul>
            </td>
            <td>
                
                <pre>
                    type = macaddr
                </pre>
                
                
            </td>
        </tr>
        
        <tr>
            <td>macaddr8</td>
            <td>macaddr8</td>
            <td>
                <ul>
                </ul>
            </td>
            <td>
                
                <pre>
                    type = macaddr8
                </pre>
                
                
            </td>
        </tr>
        
        <tr>
            <td>money</td>
            <td>money</td>
            <td>
                <ul>
                </ul>
            </td>
            <td>
                
                <pre>
                    type = money
                </pre>
                
                
            </td>
        </tr>
        
        <tr>
            <td>numeric</td>
            <td>numeric</td>
            <td>
                <ul>
                </ul>
            </td>
            <td>
                
                <pre>
                    type = numeric
                </pre>
                
                
            </td>
        </tr>
        
        <tr>
            <td>path</td>
            <td>path</td>
            <td>
                <ul>
                </ul>
            </td>
            <td>
                
                <pre>
                    type = path
                </pre>
                
                
            </td>
        </tr>
        
        <tr>
            <td>point</td>
            <td>point</td>
            <td>
                <ul>
                </ul>
            </td>
            <td>
                
                <pre>
                    type = point
                </pre>
                
                
            </td>
        </tr>
        
        <tr>
            <td>real</td>
            <td>real</td>
            <td>
                <ul>
                </ul>
            </td>
            <td>
                
                <pre>
                    type = real
                </pre>
                
                
            </td>
        </tr>
        
        <tr>
            <td>serial</td>
            <td>serial</td>
            <td>
                <ul>
                </ul>
            </td>
            <td>
                
                <pre>
                    type = serial
                </pre>
                
                
            </td>
        </tr>
        
        <tr>
            <td>serial2</td>
            <td>serial2</td>
            <td>
                <ul>
                </ul>
            </td>
            <td>
                
                <pre>
                    type = serial2
                </pre>
                
                
            </td>
        </tr>
        
        <tr>
            <td>serial4</td>
            <td>serial4</td>
            <td>
                <ul>
                </ul>
            </td>
            <td>
                
                <pre>
                    type = serial4
                </pre>
                
                
            </td>
        </tr>
        
        <tr>
            <td>serial8</td>
            <td>serial8</td>
            <td>
                <ul>
                </ul>
            </td>
            <td>
                
                <pre>
                    type = serial8
                </pre>
                
                
            </td>
        </tr>
        
        <tr>
            <td>smallint</td>
            <td>smallint</td>
            <td>
                <ul>
                </ul>
            </td>
            <td>
                
                <pre>
                    type = smallint
                </pre>
                
                
            </td>
        </tr>
        
        <tr>
            <td>smallserial</td>
            <td>smallserial</td>
            <td>
                <ul>
                </ul>
            </td>
            <td>
                
                <pre>
                    type = smallserial
                </pre>
                
                
            </td>
        </tr>
        
        <tr>
            <td>sql</td>
            <td>sql</td>
            <td>
                <ul>
                        <li>def (string)</li>
                </ul>
            </td>
            <td>
                
                <pre>
                    type = sql("a")
                </pre>
                
                
            </td>
        </tr>
        
        <tr>
            <td>text</td>
            <td>text</td>
            <td>
                <ul>
                </ul>
            </td>
            <td>
                
                <pre>
                    type = text
                </pre>
                
                
            </td>
        </tr>
        
        <tr>
            <td>time</td>
            <td>time</td>
            <td>
                <ul>
                        <li>precision (int)</li>
                </ul>
            </td>
            <td>
                
                <pre>
                    type = time(10)
                </pre>
                
                
            </td>
        </tr>
        
        <tr>
            <td>time_with_time_zone</td>
            <td>time with time zone</td>
            <td>
                <ul>
                        <li>precision (int)</li>
                </ul>
            </td>
            <td>
                
                <pre>
                    type = time_with_time_zone(10)
                </pre>
                
                
            </td>
        </tr>
        
        <tr>
            <td>time_without_time_zone</td>
            <td>time without time zone</td>
            <td>
                <ul>
                        <li>precision (int)</li>
                </ul>
            </td>
            <td>
                
                <pre>
                    type = time_without_time_zone(10)
                </pre>
                
                
            </td>
        </tr>
        
        <tr>
            <td>timestamp</td>
            <td>timestamp</td>
            <td>
                <ul>
                        <li>precision (int)</li>
                </ul>
            </td>
            <td>
                
                <pre>
                    type = timestamp(10)
                </pre>
                
                
            </td>
        </tr>
        
        <tr>
            <td>timestamp_with_time_zone</td>
            <td>timestamp with time zone</td>
            <td>
                <ul>
                        <li>precision (int)</li>
                </ul>
            </td>
            <td>
                
                <pre>
                    type = timestamp_with_time_zone(10)
                </pre>
                
                
            </td>
        </tr>
        
        <tr>
            <td>timestamp_without_time_zone</td>
            <td>timestamp without time zone</td>
            <td>
                <ul>
                        <li>precision (int)</li>
                </ul>
            </td>
            <td>
                
                <pre>
                    type = timestamp_without_time_zone(10)
                </pre>
                
                
            </td>
        </tr>
        
        <tr>
            <td>timestamptz</td>
            <td>timestamptz</td>
            <td>
                <ul>
                        <li>precision (int)</li>
                </ul>
            </td>
            <td>
                
                <pre>
                    type = timestamptz(10)
                </pre>
                
                
            </td>
        </tr>
        
        <tr>
            <td>uuid</td>
            <td>uuid</td>
            <td>
                <ul>
                </ul>
            </td>
            <td>
                
                <pre>
                    type = uuid
                </pre>
                
                
            </td>
        </tr>
        
        <tr>
            <td>varchar</td>
            <td>varchar</td>
            <td>
                <ul>
                        <li>size (int)</li>
                </ul>
            </td>
            <td>
                
                <pre>
                    type = varchar(255)
                </pre>
                
                
            </td>
        </tr>
        
        <tr>
            <td>xml</td>
            <td>xml</td>
            <td>
                <ul>
                </ul>
            </td>
            <td>
                
                <pre>
                    type = xml
                </pre>
                
                
            </td>
        </tr>
        
    </tbody>
</table>

### SQLite

<table>
    <thead>
    <tr>
        <th>HCL</th>
        <th>SQL</th>
        <th>Attributes</th>
        <th>Example</th>
    </tr>
    </thead>
    <tbody>
        
        <tr>
            <td>bigint</td>
            <td>bigint</td>
            <td>
                <ul>
                        <li>size (int)</li>
                </ul>
            </td>
            <td>
                
                <pre>
                    type = bigint(255)
                </pre>
                
                
            </td>
        </tr>
        
        <tr>
            <td>blob</td>
            <td>blob</td>
            <td>
                <ul>
                        <li>size (int)</li>
                </ul>
            </td>
            <td>
                
                <pre>
                    type = blob(255)
                </pre>
                
                
            </td>
        </tr>
        
        <tr>
            <td>boolean</td>
            <td>boolean</td>
            <td>
                <ul>
                </ul>
            </td>
            <td>
                
                <pre>
                    type = boolean
                </pre>
                
                
            </td>
        </tr>
        
        <tr>
            <td>character</td>
            <td>character</td>
            <td>
                <ul>
                        <li>size (int)</li>
                </ul>
            </td>
            <td>
                
                <pre>
                    type = character(255)
                </pre>
                
                
            </td>
        </tr>
        
        <tr>
            <td>clob</td>
            <td>clob</td>
            <td>
                <ul>
                        <li>size (int)</li>
                </ul>
            </td>
            <td>
                
                <pre>
                    type = clob(255)
                </pre>
                
                
            </td>
        </tr>
        
        <tr>
            <td>date</td>
            <td>date</td>
            <td>
                <ul>
                </ul>
            </td>
            <td>
                
                <pre>
                    type = date
                </pre>
                
                
            </td>
        </tr>
        
        <tr>
            <td>datetime</td>
            <td>datetime</td>
            <td>
                <ul>
                </ul>
            </td>
            <td>
                
                <pre>
                    type = datetime
                </pre>
                
                
            </td>
        </tr>
        
        <tr>
            <td>decimal</td>
            <td>decimal</td>
            <td>
                <ul>
                        <li>precision (int)</li>
                        <li>scale (int)</li>
                </ul>
            </td>
            <td>
                
                <pre>
                    type = decimal(10,2)
                </pre>
                
                
            </td>
        </tr>
        
        <tr>
            <td>double</td>
            <td>double</td>
            <td>
                <ul>
                        <li>size (int)</li>
                </ul>
            </td>
            <td>
                
                <pre>
                    type = double(255)
                </pre>
                
                
            </td>
        </tr>
        
        <tr>
            <td>double_precision</td>
            <td>double precision</td>
            <td>
                <ul>
                        <li>size (int)</li>
                </ul>
            </td>
            <td>
                
                <pre>
                    type = double_precision(255)
                </pre>
                
                
            </td>
        </tr>
        
        <tr>
            <td>float</td>
            <td>float</td>
            <td>
                <ul>
                        <li>size (int)</li>
                </ul>
            </td>
            <td>
                
                <pre>
                    type = float(255)
                </pre>
                
                
            </td>
        </tr>
        
        <tr>
            <td>int</td>
            <td>int</td>
            <td>
                <ul>
                        <li>size (int)</li>
                </ul>
            </td>
            <td>
                
                <pre>
                    type = int(255)
                </pre>
                
                
            </td>
        </tr>
        
        <tr>
            <td>int2</td>
            <td>int2</td>
            <td>
                <ul>
                        <li>size (int)</li>
                </ul>
            </td>
            <td>
                
                <pre>
                    type = int2(255)
                </pre>
                
                
            </td>
        </tr>
        
        <tr>
            <td>int8</td>
            <td>int8</td>
            <td>
                <ul>
                        <li>size (int)</li>
                </ul>
            </td>
            <td>
                
                <pre>
                    type = int8(255)
                </pre>
                
                
            </td>
        </tr>
        
        <tr>
            <td>integer</td>
            <td>integer</td>
            <td>
                <ul>
                        <li>size (int)</li>
                </ul>
            </td>
            <td>
                
                <pre>
                    type = integer(255)
                </pre>
                
                
            </td>
        </tr>
        
        <tr>
            <td>json</td>
            <td>json</td>
            <td>
                <ul>
                </ul>
            </td>
            <td>
                
                <pre>
                    type = json
                </pre>
                
                
            </td>
        </tr>
        
        <tr>
            <td>mediumint</td>
            <td>mediumint</td>
            <td>
                <ul>
                        <li>size (int)</li>
                </ul>
            </td>
            <td>
                
                <pre>
                    type = mediumint(255)
                </pre>
                
                
            </td>
        </tr>
        
        <tr>
            <td>native_character</td>
            <td>native character</td>
            <td>
                <ul>
                        <li>size (int)</li>
                </ul>
            </td>
            <td>
                
                <pre>
                    type = native_character(255)
                </pre>
                
                
            </td>
        </tr>
        
        <tr>
            <td>nchar</td>
            <td>nchar</td>
            <td>
                <ul>
                        <li>size (int)</li>
                </ul>
            </td>
            <td>
                
                <pre>
                    type = nchar(255)
                </pre>
                
                
            </td>
        </tr>
        
        <tr>
            <td>numeric</td>
            <td>numeric</td>
            <td>
                <ul>
                        <li>precision (int)</li>
                        <li>scale (int)</li>
                </ul>
            </td>
            <td>
                
                <pre>
                    type = numeric(10,2)
                </pre>
                
                
            </td>
        </tr>
        
        <tr>
            <td>nvarchar</td>
            <td>nvarchar</td>
            <td>
                <ul>
                        <li>size (int)</li>
                </ul>
            </td>
            <td>
                
                <pre>
                    type = nvarchar(255)
                </pre>
                
                
            </td>
        </tr>
        
        <tr>
            <td>real</td>
            <td>real</td>
            <td>
                <ul>
                        <li>precision (int)</li>
                        <li>scale (int)</li>
                </ul>
            </td>
            <td>
                
                <pre>
                    type = real(10,2)
                </pre>
                
                
            </td>
        </tr>
        
        <tr>
            <td>smallint</td>
            <td>smallint</td>
            <td>
                <ul>
                        <li>size (int)</li>
                </ul>
            </td>
            <td>
                
                <pre>
                    type = smallint(255)
                </pre>
                
                
            </td>
        </tr>
        
        <tr>
            <td>text</td>
            <td>text</td>
            <td>
                <ul>
                        <li>size (int)</li>
                </ul>
            </td>
            <td>
                
                <pre>
                    type = text(255)
                </pre>
                
                
            </td>
        </tr>
        
        <tr>
            <td>tinyint</td>
            <td>tinyint</td>
            <td>
                <ul>
                        <li>size (int)</li>
                </ul>
            </td>
            <td>
                
                <pre>
                    type = tinyint(255)
                </pre>
                
                
            </td>
        </tr>
        
        <tr>
            <td>unsigned_big_int</td>
            <td>unsigned big int</td>
            <td>
                <ul>
                        <li>size (int)</li>
                </ul>
            </td>
            <td>
                
                <pre>
                    type = unsigned_big_int(255)
                </pre>
                
                
            </td>
        </tr>
        
        <tr>
            <td>uuid</td>
            <td>uuid</td>
            <td>
                <ul>
                </ul>
            </td>
            <td>
                
                <pre>
                    type = uuid
                </pre>
                
                
            </td>
        </tr>
        
        <tr>
            <td>varchar</td>
            <td>varchar</td>
            <td>
                <ul>
                        <li>size (int)</li>
                </ul>
            </td>
            <td>
                
                <pre>
                    type = varchar(255)
                </pre>
                
                
            </td>
        </tr>
        
        <tr>
            <td>varying_character</td>
            <td>varying character</td>
            <td>
                <ul>
                        <li>size (int)</li>
                </ul>
            </td>
            <td>
                
                <pre>
                    type = varying_character(255)
                </pre>
                
                
            </td>
        </tr>
        
    </tbody>
</table>



