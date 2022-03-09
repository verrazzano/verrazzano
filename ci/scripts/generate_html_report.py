# Copyright (c) 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

import sys
from os.path import exists


def get_vulnerability_url(id):
    if (id.lower().startswith("cve")) :
        return 'https://nvd.nist.gov/vuln/detail/'+ id
    elif (id.lower().startswith("elsa")):
        return 'https://linux.oracle.com/errata/'+ id + '.html'
    elif (id.lower().startswith("ghsa")):
        return 'https://github.com/advisories/' + id
    elif (id.lower().startswith("go")):
        return 'https://osv.dev/vulnerability/' + id
    else:
        return id


def get_vulnerability_anchor(id):
    return '<a href="' + get_vulnerability_url(id) + '" target="_blank">'+id+'</a>'


def write_table_header(headers, html_file):
    heading = "<thead>\n" + "<tr>\n"
    for header in headers:
        heading += "<th>\n" + header + "</th>\n"
    heading += "</tr>\n </thead>\n"
    html_file.write(heading)


def write_table_body(csv_file_path, html_file):
    if not exists(csv_file_path):
        print("[WARN] CSV file '%s' does not exist" % csv_file_path)
        return
    csv_file = open(csv_file_path, 'r')
    body = "<tbody>\n"
    lineCount = 0
    while True:
        csv_line = csv_file.readline()
        if not csv_line:
            break
        row_data = csv_line.split(',')
        body += "<tr>\n"
        body += "<td>\n" + get_vulnerability_anchor(row_data[6]) + "\n</td>\n"
        body += "<td>" + row_data[5] + "</td>\n"
        body += "<td>" + row_data[7] + "</td>\n"
        body += "<td>" + row_data[8] + "</td>\n"
        body += "</tr>\n"
        lineCount += 1
    print("Processed %d lines" % lineCount)
    body += "</tbody>\n"
    html_file.write(body)


def write_csv_to_html(headers, csv_file_path, html_dir):
    if not exists(html_dir):
        print("[WARN] Directory to write html report '%s' does not exist" % html_dir)
        return
    html_file_path = html_dir + "/consolidated-scan-report.html"
    html_file = open(html_file_path, 'w')
    html_file.write("<table>\n")
    write_table_header(headers, html_file)
    write_table_body(csv_file_path, html_file)
    html_file.write("</table>\n")
    html_file.close()

# headers for table
headers = ["Vulnerability", "Scan Tool", "Severity", "Artifact"]
csv_file_path=""
html_report_path=""

if len(sys.argv) < 2:
   print("Missing argument for csv file")
   exit(1)
else:
    csv_file_path = sys.argv[1]

if len(sys.argv) < 3:
    print("Missing argument for html file")
    exit(1)
else:
    html_report_path = sys.argv[2]

write_csv_to_html(headers, csv_file_path, html_report_path)

