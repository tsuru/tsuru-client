import re
import os
import json
from subprocess import check_output

import docutils
from docutils import nodes
from docutils.parsers import rst
from docutils.parsers.rst.directives import unchanged


class CommandNode(nodes.Element):
    pass


class CommandDirective(rst.Directive):
    has_content = False
    final_argument_whitespace = True
    required_arguments = 1
    option_spec = dict(title=unchanged)

    def run(self):
        node = CommandNode()
        node.line = self.lineno
        node['command'] = self.arguments[0]
        node['title'] = self.options.get('title', '')
        return [node]


def run_programs(app, doctree):

    for node in doctree.traverse(CommandNode):
        command = node['command']
        try:
            command_data = app.env.tsuru_json[command]
        except KeyError:
            error_message = 'Command {0} not found'.format(command)
            error_node = doctree.reporter.error(error_message, base_node=node)
            node.replace_self(error_node)
            continue
        topic = command_data.get('topic')
        if topic:
            render_topic(app, node, topic)
        else:
            render_cmd(app, node, command_data['usage'], command_data['desc'])


def render_topic(app, node, topic):
    paragraph = nodes.paragraph('', topic)
    node.replace_self(paragraph)


idregex = re.compile(r'[^a-zA-Z0-9]')
inline_literal_regex = re.compile(r'\[\[|\]\]')


def render_cmd(app, node, usage, description):
    title = node.get('title')

    titleid = idregex.sub('-', title).lower()
    section = nodes.section('', ids=[titleid])

    if title:
        section.append(nodes.title(title, title))

    output = "$ {}".format(usage)
    new_node = nodes.literal_block(output, output)
    new_node['language'] = 'text'
    section.append(new_node)

    settings = docutils.frontend.OptionParser(
        components=(docutils.parsers.rst.Parser,)
    ).get_default_values()
    document = docutils.utils.new_document('', settings)
    parser = docutils.parsers.rst.Parser()
    description = inline_literal_regex.sub('``', description)
    parser.parse(description, document)
    for el in document.children:
        section.append(el)

    node.replace_self(section)


def read_cmds(app):
    if not hasattr(app.env, 'tsuru_json'):
        destination = os.path.join(os.path.dirname(os.path.abspath(__file__)), 'cmds.json')
        with open(destination, 'r') as f:
            data = json.loads(f.read())
        app.env.tsuru_json = data


def setup(app):
    app.add_directive('tsuru-command', CommandDirective)
    app.connect(str('builder-inited'), read_cmds)
    app.connect(str('doctree-read'), run_programs)


def main():
    parts_regex = re.compile(r'tsuru version.*Usage: (.*?)\n+(.*)', re.DOTALL)
    topic_regex = re.compile(r'tsuru version.*?\n+(.*)\n\n.*?\n\n  ', re.DOTALL)

    result = check_output("tsuru | egrep \"^[  ]\" | awk -F ' ' '{print $1}'", shell=True)
    cmds = result.split('\n')
    final_result = {}
    for cmd in cmds:
        if not cmd:
            continue
        result = check_output('tsuru help {}'.format(cmd), shell=True)
        matchdata = parts_regex.match(result)
        if matchdata is None:
            topicdata = topic_regex.match(result)
            if topicdata is None:
                print "Ignored command: {}".format(cmd)
                continue
            result = {
                'topic': topicdata.group(1)
            }
        else:
            result = {
                'usage': matchdata.group(1),
                'desc': matchdata.group(2),
            }
        final_result[cmd] = result
    out = json.dumps(final_result, indent=2)
    destination = os.path.join(os.path.dirname(os.path.abspath(__file__)), 'cmds.json')
    with open(destination, 'wb') as f:
        f.write(out)


if __name__ == '__main__':
    main()
