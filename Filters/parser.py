import urllib2, datetime, mmap, re, os, sys

## GLOBAL VAR ##
dir = os.path.dirname(__file__)
processed_rules = set()
exclusions_file = open(os.path.join(dir, 'exclusions.txt'), 'r').read().split('\n')
# Remove comments
exclusions = filter(lambda line : not line.startswith('!'), exclusions_file)
  
## FUNCTION ##
def is_domain_rule(rule):
    point_idx = rule.find('.')
    if point_idx == -1:
        return False
    question_idx = rule.find('?', point_idx);    
    slash_idx = rule.find('/', point_idx)
    if slash_idx == -1 and question_idx == -1:
        return True
    replace_idx =  slash_idx if slash_idx != -1 else question_idx
    tail = rule[replace_idx:]
    return len(tail) <= 2

def date_now():
    return datetime.datetime.now().strftime("%Y-%m-%d %H:%M:%S") 

def get_content(url):
    r = urllib2.urlopen(url)    
    return r.read().split('\n')

def save_comment(comment, f):
    idx = comment.find('%timestamp%')
    if idx != -1:
       comment = comment[:idx] + date_now() + '\n'
    f.writelines(comment)    

def is_rule_not_exclusion(rule):
    for line in exclusions:
        if line in rule and line != '':  
            return False        
    return True

def is_not_duplication(rule):
    return rule not in processed_rules
  
def write_rule(rule, f):
    if (is_domain_rule(rule) and is_not_duplication(rule)) or rule.startswith('!'):
        f.writelines(rule + '\n')
        processed_rules.add(rule)

def save_url_rule(line, f):
    url = line.replace('url', '').strip() 
    for rule in get_content(url):
        if is_rule_not_exclusion(rule):      
            if rule.find('$') != -1:
                idx = rule.find('$');
                write_rule(rule[:idx], f)        
            else: 
                write_rule(rule, f)

def save_file_rule(line, f):
    file_name = line.replace('file', '').strip()
    with open(os.path.join(dir, file_name), 'r') as rf:
        for rule in rf:
            write_rule(rule.rstrip(), f)

## MAIN ##
with open(os.path.join(dir, 'filter.template'), 'r') as tmpl:    
   with open(os.path.join(dir, 'filter.txt'), 'w') as f:
       for line in tmpl:
            if line.startswith('!'):
                save_comment(line, f)
            if line.startswith('url'):
                save_url_rule(line, f)
            if line.startswith('file'):
                save_file_rule(line, f)              
sys.exit(0)

